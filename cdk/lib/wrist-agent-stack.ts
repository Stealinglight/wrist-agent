import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as ssm from 'aws-cdk-lib/aws-ssm';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as iam from 'aws-cdk-lib/aws-iam';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import * as bedrock from '@aws-cdk/aws-bedrock-alpha';
import { Construct } from 'constructs';

// Configuration constants
const THROTTLE_RATE_LIMIT = 10;
const THROTTLE_BURST_LIMIT = 20;
const TOKEN_CACHE_TTL_SECONDS = 300; // 5 minutes

export interface StackConfig {
  region: string;
  modelId: string;
  geoRegion: 'US' | 'EU';
  clientTokenParamName: string;
  clientTokenValue: string;
}

export interface WristAgentStackProps extends cdk.StackProps {
  config: StackConfig;
}

export class WristAgentStack extends cdk.Stack {
  public readonly fn: lambda.Function;
  public readonly authorizerFn: lambda.Function;
  public readonly api: apigateway.RestApi;

  constructor(scope: Construct, id: string, props: WristAgentStackProps) {
    super(scope, id, props);

    const { config } = props;

    // Create cross-region inference profile for Claude Haiku 4.5
    const crossRegionProfile = bedrock.CrossRegionInferenceProfile.fromConfig({
      geoRegion: config.geoRegion === 'US'
        ? bedrock.CrossRegionInferenceProfileRegion.US
        : bedrock.CrossRegionInferenceProfileRegion.EU,
      model: bedrock.BedrockFoundationModel.ANTHROPIC_CLAUDE_HAIKU_4_5_V1_0,
    });

    // Create SSM parameter for client token
    // NOTE: CDK creates this as a StringParameter (unencrypted) because SecureString
    // values cannot be created via CloudFormation (the value would be exposed in templates).
    // After deployment, users SHOULD convert this to SecureString using:
    //   aws ssm put-parameter --name "/wrist-agent/client-token" --value "YOUR_TOKEN" --type SecureString --overwrite
    // The authorizer uses WithDecryption: true and works with both String and SecureString.
    const tokenParam = new ssm.StringParameter(this, 'ClientToken', {
      parameterName: config.clientTokenParamName,
      stringValue: config.clientTokenValue,
      description: 'Shared client token for Wrist Agent requests. Convert to SecureString after deployment for production use.',
      tier: ssm.ParameterTier.STANDARD,
    });

    // Create Lambda Authorizer function
    this.authorizerFn = new GoFunction(this, 'WristAgentAuthorizer', {
      entry: '../lambda-authorizer',
      architecture: lambda.Architecture.ARM_64,
      runtime: lambda.Runtime.PROVIDED_AL2,
      timeout: cdk.Duration.seconds(10),
      memorySize: 128,
      environment: {
        CLIENT_TOKEN_PARAM_NAME: config.clientTokenParamName,
        TOKEN_CACHE_TTL_SECONDS: String(TOKEN_CACHE_TTL_SECONDS),
      },
      description: 'Wrist Agent API Gateway Lambda Authorizer',
    });

    // Grant authorizer function access to SSM parameter (both String and SecureString)
    tokenParam.grantRead(this.authorizerFn);

    // Grant KMS decrypt permission for SecureString parameters
    // Scoped to the AWS-managed SSM key (alias/aws/ssm) for least-privilege
    this.authorizerFn.addToRolePolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: ['kms:Decrypt'],
      resources: [
        `arn:aws:kms:${config.region}:${this.account}:alias/aws/ssm`,
      ],
      conditions: {
        StringEquals: {
          'kms:ViaService': `ssm.${config.region}.amazonaws.com`,
        },
      },
    }));

    // Create main handler Lambda function
    this.fn = new GoFunction(this, 'WristAgentHandler', {
      entry: '../lambda',
      architecture: lambda.Architecture.ARM_64,
      runtime: lambda.Runtime.PROVIDED_AL2,
      timeout: cdk.Duration.seconds(300),
      memorySize: 256,
      environment: {
        BEDROCK_REGION: config.region,
        BEDROCK_MODEL_ID: crossRegionProfile.inferenceProfileId,
      },
      description: 'Wrist Agent Lambda handler for Bedrock integration',
    });

    // Grant cross-region inference permissions
    crossRegionProfile.grantInvoke(this.fn);

    // Create REST API with logging
    const logGroup = new logs.LogGroup(this, 'ApiGatewayLogs', {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    this.api = new apigateway.RestApi(this, 'WristAgentApi', {
      restApiName: 'Wrist Agent API',
      description: 'API Gateway for Wrist Agent - Apple Watch to Bedrock integration',
      deployOptions: {
        stageName: 'prod',
        accessLogDestination: new apigateway.LogGroupLogDestination(logGroup),
        accessLogFormat: apigateway.AccessLogFormat.jsonWithStandardFields({
          caller: true,
          httpMethod: true,
          ip: true,
          protocol: true,
          requestTime: true,
          resourcePath: true,
          responseLength: true,
          status: true,
          user: true,
        }),
        loggingLevel: apigateway.MethodLoggingLevel.INFO,
        dataTraceEnabled: false,
        throttlingRateLimit: THROTTLE_RATE_LIMIT,
        throttlingBurstLimit: THROTTLE_BURST_LIMIT,
      },
      // CORS Configuration: Using wildcard origins because:
      // 1. Apple Shortcuts don't use traditional browser-based CORS
      // 2. Primary security is provided by token-based authentication
      // 3. API Gateway handles CORS preflight - Lambda doesn't need CORS headers
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: ['POST', 'OPTIONS'],
        allowHeaders: ['Content-Type', 'X-Client-Token'],
        maxAge: cdk.Duration.hours(1),
      },
      endpointTypes: [apigateway.EndpointType.REGIONAL],
    });

    // Create REQUEST type Lambda Authorizer
    const authorizer = new apigateway.RequestAuthorizer(this, 'TokenAuthorizer', {
      handler: this.authorizerFn,
      identitySources: [apigateway.IdentitySource.header('X-Client-Token')],
      resultsCacheTtl: cdk.Duration.seconds(TOKEN_CACHE_TTL_SECONDS),
      authorizerName: 'WristAgentTokenAuthorizer',
    });

    // Create /invoke resource with POST method
    const invokeResource = this.api.root.addResource('invoke');
    invokeResource.addMethod('POST', new apigateway.LambdaIntegration(this.fn, {
      proxy: true,
      allowTestInvoke: true,
    }), {
      authorizer: authorizer,
      authorizationType: apigateway.AuthorizationType.CUSTOM,
    });

    // Output the API Gateway URL
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: this.api.url,
      description: 'API Gateway endpoint URL for Wrist Agent',
      exportName: 'WristAgentApiEndpoint',
    });

    new cdk.CfnOutput(this, 'InvokeEndpoint', {
      value: `${this.api.url}invoke`,
      description: 'Full invoke endpoint URL for Wrist Agent',
      exportName: 'WristAgentInvokeEndpoint',
    });

    new cdk.CfnOutput(this, 'TokenParameterName', {
      value: tokenParam.parameterName,
      description: 'SSM parameter name for the Wrist Agent client token',
      exportName: 'WristAgentTokenParameterName',
    });

    new cdk.CfnOutput(this, 'InferenceProfileId', {
      value: crossRegionProfile.inferenceProfileId,
      description: 'Cross-region inference profile ID for Bedrock',
      exportName: 'WristAgentInferenceProfileId',
    });

    // Add tags to all resources
    cdk.Tags.of(this).add('Project', 'WristAgent');
    cdk.Tags.of(this).add('Component', 'Infrastructure');
  }
}
