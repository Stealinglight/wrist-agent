import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as ssm from 'aws-cdk-lib/aws-ssm';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import { Construct } from 'constructs';

export interface StackConfig {
  region: string;
  modelId: string;
  clientTokenParamName: string;
  clientTokenValue: string;
}

export interface WristAgentStackProps extends cdk.StackProps {
  config: StackConfig;
}

export class WristAgentStack extends cdk.Stack {
  public readonly fn: lambda.Function;
  public readonly fUrl: lambda.FunctionUrl;

  constructor(scope: Construct, id: string, props: WristAgentStackProps) {
    super(scope, id, props);

    const { config } = props;

    // Create Go Lambda function
    this.fn = new GoFunction(this, 'WristAgentHandler', {
      entry: '../lambda',
      architecture: lambda.Architecture.ARM_64,
      runtime: lambda.Runtime.PROVIDED_AL2,
      timeout: cdk.Duration.seconds(300),
      memorySize: 256,
      environment: {
        BEDROCK_REGION: config.region,
        BEDROCK_MODEL_ID: config.modelId,
        CLIENT_TOKEN_PARAM_NAME: config.clientTokenParamName,
      },
      description: 'Wrist Agent Lambda handler for Bedrock integration',
    });

    const tokenParam = new ssm.StringParameter(this, 'ClientToken', {
      parameterName: config.clientTokenParamName,
      stringValue: config.clientTokenValue,
      description: 'Shared client token for Wrist Agent requests',
      tier: ssm.ParameterTier.STANDARD,
    });

    this.fn.addToRolePolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        actions: ['bedrock:InvokeModel'],
        resources: [`arn:aws:bedrock:${config.region}::foundation-model/${config.modelId}`],
      })
    );

    tokenParam.grantRead(this.fn);

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

    new cdk.CfnOutput(this, 'TokenParameterName', {
      value: tokenParam.parameterName,
      description: 'SSM parameter name for the Wrist Agent client token',
      exportName: 'WristAgentTokenParameterName',
    });

    // Add tags to all resources
    cdk.Tags.of(this).add('Project', 'WristAgent');
    cdk.Tags.of(this).add('Component', 'Infrastructure');
  }
}
