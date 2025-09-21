import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as ssm from 'aws-cdk-lib/aws-ssm';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import { Construct } from 'constructs';

export interface StackConfig {
  region: string;
  modelId: string;
  clientTokenParam: string;
}

export interface WristAgentStackProps extends cdk.StackProps {
  config: StackConfig;
}

export class WristAgentStack extends cdk.Stack {
  public readonly tokenParam: ssm.StringParameter;
  public readonly fn: lambda.Function;
  public readonly fUrl: lambda.FunctionUrl;

  constructor(scope: Construct, id: string, props: WristAgentStackProps) {
    super(scope, id, props);

    const { config } = props;

    // Create SSM Parameter for client token storage
    this.tokenParam = new ssm.StringParameter(this, 'ClientToken', {
      parameterName: config.clientTokenParam,
      description: 'Authentication token for Wrist Agent API access',
      stringValue: 'CHANGE_ME_' + Math.random().toString(36).substring(2, 15),
      tier: ssm.ParameterTier.STANDARD,
    });

    // Create encrypted SSM Parameter for Bedrock API key (if provided)
    let bedrockApiKeyParam: ssm.StringParameter | undefined;
    if (process.env.BEDROCK_API_KEY) {
      bedrockApiKeyParam = new ssm.StringParameter(this, 'BedrockAPIKey', {
        parameterName: '/wrist-agent/bedrock-api-key',
        description: 'Encrypted Bedrock API key for authentication',
        stringValue: process.env.BEDROCK_API_KEY,
        type: ssm.ParameterType.SECURE_STRING, // Explicitly create as encrypted SecureString
        tier: ssm.ParameterTier.STANDARD,
      });
    }

    // Create Go Lambda function
    this.fn = new GoFunction(this, 'WristAgentHandler', {
      entry: '../lambda',
      architecture: lambda.Architecture.ARM_64,
      runtime: lambda.Runtime.PROVIDED_AL2,
      timeout: cdk.Duration.seconds(30),
      memorySize: 256,
      environment: {
        BEDROCK_REGION: config.region,
        MODEL_ID: config.modelId,
        CLIENT_TOKEN_PARAM: config.clientTokenParam,
        // SSM parameter name for encrypted Bedrock API key
        BEDROCK_API_KEY_PARAM: '/wrist-agent/bedrock-api-key',
        // For local development, BEDROCK_API_KEY can be set directly
        ...(process.env.BEDROCK_API_KEY &&
          !bedrockApiKeyParam && {
            BEDROCK_API_KEY: process.env.BEDROCK_API_KEY,
          }),
      },
      description: 'Wrist Agent Lambda handler for Bedrock integration with hybrid authentication',
    });

    // Note: Bedrock IAM permissions removed - using static API key authentication instead
    // If you need to fall back to IAM role authentication, uncomment the following:
    // this.fn.addToRolePolicy(
    //   new iam.PolicyStatement({
    //     effect: iam.Effect.ALLOW,
    //     actions: ['bedrock:InvokeModel'],
    //     resources: [`arn:aws:bedrock:${config.region}::foundation-model/${config.modelId}`],
    //   })
    // );

    // Grant SSM parameter read access
    this.tokenParam.grantRead(this.fn);

    // Grant access to encrypted Bedrock API key parameter if it exists
    if (bedrockApiKeyParam) {
      bedrockApiKeyParam.grantRead(this.fn);
    }

    // Create Function URL with CORS configuration
    this.fUrl = this.fn.addFunctionUrl({
      authType: lambda.FunctionUrlAuthType.NONE,
      cors: {
        allowCredentials: false,
        allowedHeaders: ['Content-Type', 'X-Client-Token'],
        allowedMethods: [lambda.HttpMethod.POST, lambda.HttpMethod.OPTIONS],
        allowedOrigins: ['*'],
        maxAge: cdk.Duration.hours(1),
      },
    });

    // Output the Function URL
    new cdk.CfnOutput(this, 'FunctionUrl', {
      value: this.fUrl.url,
      description: 'Lambda Function URL for Wrist Agent API',
      exportName: 'WristAgentFunctionUrl',
    });

    // Output the SSM parameter name
    new cdk.CfnOutput(this, 'TokenParameterName', {
      value: this.tokenParam.parameterName,
      description: 'SSM Parameter name for client token',
      exportName: 'WristAgentTokenParam',
    });

    // Add tags to all resources
    cdk.Tags.of(this).add('Project', 'WristAgent');
    cdk.Tags.of(this).add('Component', 'Infrastructure');
  }
}
