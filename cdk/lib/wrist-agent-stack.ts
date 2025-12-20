import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as ssm from 'aws-cdk-lib/aws-ssm';
import * as logs from 'aws-cdk-lib/aws-logs';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import * as bedrock from '@aws-cdk/aws-bedrock-alpha';
import { Construct } from 'constructs';

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
    const tokenParam = new ssm.StringParameter(this, 'ClientToken', {
      parameterName: config.clientTokenParamName,
      stringValue: config.clientTokenValue,
      description: 'Shared client token for Wrist Agent requests',
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
      },
      description: 'Wrist Agent API Gateway Lambda Authorizer',
    });

    // Grant authorizer function access to SSM parameter
    tokenParam.grantRead(this.authorizerFn);

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
        throttlingRateLimit: 10,
        throttlingBurstLimit: 20,
      },
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
      resultsCacheTtl: cdk.Duration.minutes(5),
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
