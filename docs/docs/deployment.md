---
sidebar_position: 7
---

# Deployment Guide

This guide covers deployment strategies, best practices, and production considerations for your Wrist Agent system.

## Deployment Methods

### Local Deployment

Deploy directly from your development machine using AWS CDK.

**Prerequisites:**
- AWS CLI configured with appropriate credentials
- Node.js 18+ and npm installed
- Go 1.22+ installed

**Steps:**

```bash
# Clone and setup
git clone https://github.com/your-username/wrist-agent.git
cd wrist-agent

# Install dependencies
cd cdk && npm install && cd ..
cd lambda && go mod tidy && cd ..

# Bootstrap CDK (first time only)
cd cdk
npx cdk bootstrap

# Deploy
npx cdk deploy --require-approval never

# Note the outputs
```

**When to use:**
- Initial setup and testing
- Development iterations
- Personal deployments
- Quick prototyping

### GitHub Actions Deployment

Automated deployment using GitHub Actions with OIDC authentication.

**Prerequisites:**
- GitHub repository (fork or your own)
- AWS OIDC provider configured
- IAM role for GitHub Actions

**Setup:**

```bash
# 1. Create OIDC provider (one-time)
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1

# 2. Create IAM role for GitHub Actions
cat > github-trust-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:YOUR_GITHUB_USERNAME/wrist-agent:*"
        }
      }
    }
  ]
}
EOF

# Replace ACCOUNT_ID and YOUR_GITHUB_USERNAME
aws iam create-role \
  --role-name WristAgentGitHubActions \
  --assume-role-policy-document file://github-trust-policy.json

# 3. Attach permissions
aws iam attach-role-policy \
  --role-name WristAgentGitHubActions \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess

# 4. Configure GitHub secrets
# Repository → Settings → Secrets and variables → Actions
# Add secret: AWS_ROLE_ARN = arn:aws:iam::ACCOUNT_ID:role/WristAgentGitHubActions
```

**Workflow:**

```yaml
# .github/workflows/deploy.yml
name: Deploy to AWS

on:
  push:
    branches: [main]

permissions:
  id-token: write
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-west-2
      
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Deploy CDK
        run: |
          cd cdk
          npm ci
          npx cdk bootstrap --require-approval never
          npx cdk deploy --require-approval never
```

**When to use:**
- Production deployments
- Team environments
- Continuous delivery
- Automated testing

## Environment Configuration

### Development Environment

Optimized for fast iteration and debugging.

```bash
# .env.development
AWS_REGION=us-west-2
AWS_PROFILE=dev
BEDROCK_MODEL_ID=anthropic.claude-haiku-4-5-20251001-v1:0
CLIENT_TOKEN_PARAM_NAME=/wrist-agent/dev/client-token
LAMBDA_TIMEOUT=60
LAMBDA_MEMORY=512
LOG_LEVEL=DEBUG
```

**Characteristics:**
- Extended timeouts for debugging
- Debug logging enabled
- Separate SSM parameter namespace
- Lower cost priority over performance

### Staging Environment

Production-like environment for testing.

```bash
# .env.staging
AWS_REGION=us-west-2
AWS_PROFILE=staging
BEDROCK_MODEL_ID=anthropic.claude-haiku-4-5-20251001-v1:0
CLIENT_TOKEN_PARAM_NAME=/wrist-agent/staging/client-token
LAMBDA_TIMEOUT=30
LAMBDA_MEMORY=256
LOG_LEVEL=INFO
```

**Characteristics:**
- Production-equivalent configuration
- Same region as production
- Separate authentication tokens
- Full monitoring enabled

### Production Environment

Optimized for reliability and performance.

```bash
# .env.production
AWS_REGION=us-west-2
AWS_PROFILE=production
BEDROCK_MODEL_ID=anthropic.claude-haiku-4-5-20251001-v1:0
CLIENT_TOKEN_PARAM_NAME=/wrist-agent/client-token
LAMBDA_TIMEOUT=30
LAMBDA_MEMORY=256
LOG_LEVEL=INFO
ENABLE_XRAY=true
```

**Characteristics:**
- Optimized memory allocation
- Production timeout limits
- X-Ray tracing enabled
- CloudWatch alarms configured

## Multi-Region Deployment

Deploy to multiple regions for redundancy or regional requirements.

### Primary Region (us-west-2)

```bash
export AWS_REGION=us-west-2
cd cdk
npx cdk deploy --context region=us-west-2
```

### Secondary Region (eu-west-1)

```bash
export AWS_REGION=eu-west-1
cd cdk
npx cdk bootstrap
npx cdk deploy --context region=eu-west-1
```

### Region Selection Considerations

**us-west-2 (Oregon):**
- ✅ Full Bedrock model availability
- ✅ Lower latency for US West Coast
- ✅ Typically lower costs

**us-east-1 (N. Virginia):**
- ✅ Full Bedrock model availability
- ✅ Lower latency for US East Coast
- ✅ Most services available first

**eu-west-1 (Ireland):**
- ✅ GDPR compliance
- ✅ Lower latency for Europe
- ⚠️ Check Bedrock model availability

## Blue-Green Deployment

Zero-downtime deployment strategy.

```typescript
// cdk/lib/blue-green-stack.ts
import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';

export class BlueGreenStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Blue version (current)
    const blueFunction = new lambda.Function(this, 'BlueFunction', {
      // ... configuration
    });

    // Green version (new)
    const greenFunction = new lambda.Function(this, 'GreenFunction', {
      // ... configuration
    });

    // Alias for gradual traffic shifting
    const alias = new lambda.Alias(this, 'LiveAlias', {
      aliasName: 'live',
      version: blueFunction.currentVersion,
    });

    // CodeDeploy for traffic shifting
    new codedeploy.LambdaDeploymentGroup(this, 'DeploymentGroup', {
      alias: alias,
      deploymentConfig: codedeploy.LambdaDeploymentConfig.LINEAR_10PERCENT_EVERY_1MINUTE,
    });
  }
}
```

## Infrastructure as Code Best Practices

### Version Control

```bash
# Tag releases
git tag -a v1.0.0 -m "Production release 1.0.0"
git push origin v1.0.0

# Deploy specific version
git checkout v1.0.0
cd cdk
npx cdk deploy
```

### CDK Stack Organization

```typescript
// cdk/bin/app.ts
import * as cdk from 'aws-cdk-lib';
import { WristAgentStack } from '../lib/wrist-agent-stack';
import { MonitoringStack } from '../lib/monitoring-stack';

const app = new cdk.App();

// Main application stack
const appStack = new WristAgentStack(app, 'WristAgentStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION,
  },
});

// Monitoring stack (separate lifecycle)
new MonitoringStack(app, 'WristAgentMonitoring', {
  functionArn: appStack.function.functionArn,
});
```

### Parameter Management

```typescript
// Use SSM for configuration
import * as ssm from 'aws-cdk-lib/aws-ssm';

const config = {
  modelId: ssm.StringParameter.valueFromLookup(
    this,
    '/wrist-agent/config/model-id'
  ),
  timeout: Number(
    ssm.StringParameter.valueFromLookup(
      this,
      '/wrist-agent/config/timeout'
    )
  ),
};
```

## Monitoring and Observability

### CloudWatch Dashboards

```typescript
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';

const dashboard = new cloudwatch.Dashboard(this, 'WristAgentDashboard', {
  dashboardName: 'WristAgent',
});

dashboard.addWidgets(
  new cloudwatch.GraphWidget({
    title: 'Lambda Invocations',
    left: [fn.metricInvocations()],
  }),
  new cloudwatch.GraphWidget({
    title: 'Error Rate',
    left: [fn.metricErrors()],
  }),
  new cloudwatch.GraphWidget({
    title: 'Duration',
    left: [fn.metricDuration()],
  })
);
```

### Alarms

```typescript
// Error rate alarm
fn.metricErrors().createAlarm(this, 'ErrorAlarm', {
  threshold: 5,
  evaluationPeriods: 2,
  alarmDescription: 'Lambda errors exceed threshold',
});

// Duration alarm
fn.metricDuration().createAlarm(this, 'DurationAlarm', {
  threshold: 25000, // 25 seconds
  evaluationPeriods: 2,
  alarmDescription: 'Lambda duration approaching timeout',
});
```

### X-Ray Tracing

```typescript
import * as lambda from 'aws-cdk-lib/aws-lambda';

const fn = new lambda.Function(this, 'Handler', {
  // ... other config
  tracing: lambda.Tracing.ACTIVE,
});
```

## Security Hardening

### Network Isolation (Optional)

Deploy Lambda in VPC for enhanced security:

```typescript
import * as ec2 from 'aws-cdk-lib/aws-ec2';

const vpc = new ec2.Vpc(this, 'VPC', {
  maxAzs: 2,
  natGateways: 1,
});

const fn = new lambda.Function(this, 'Handler', {
  // ... other config
  vpc: vpc,
  vpcSubnets: { subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS },
});
```

### Secret Rotation

Automate token rotation with Lambda:

```typescript
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';

const rotationFn = new lambda.Function(this, 'TokenRotation', {
  runtime: lambda.Runtime.NODEJS_18_X,
  handler: 'index.handler',
  code: lambda.Code.fromAsset('lambda/rotation'),
});

// Rotate monthly
new events.Rule(this, 'RotationSchedule', {
  schedule: events.Schedule.rate(cdk.Duration.days(30)),
  targets: [new targets.LambdaFunction(rotationFn)],
});
```

### Resource Policies

```typescript
// Restrict Lambda Function URL access
import * as iam from 'aws-cdk-lib/aws-iam';

fn.addPermission('FunctionUrlInvoke', {
  principal: new iam.AnyPrincipal(),
  action: 'lambda:InvokeFunctionUrl',
  functionUrlAuthType: lambda.FunctionUrlAuthType.NONE,
});
```

## Cost Optimization

### Right-Size Lambda

```bash
# Test different memory configurations
for memory in 128 256 512 1024; do
  echo "Testing with ${memory}MB memory"
  # Update Lambda memory
  aws lambda update-function-configuration \
    --function-name WristAgentStack-WristAgentHandler \
    --memory-size $memory
  
  # Run load test
  # Measure duration and cost
done
```

### Reserved Capacity

For consistent usage, consider reserved capacity:

```typescript
// Add provisioned concurrency during peak hours
import * as applicationautoscaling from 'aws-cdk-lib/aws-applicationautoscaling';

const alias = fn.currentVersion.addAlias('live');

const target = new applicationautoscaling.ScalableTarget(this, 'ScalableTarget', {
  serviceNamespace: applicationautoscaling.ServiceNamespace.LAMBDA,
  maxCapacity: 10,
  minCapacity: 1,
  resourceId: `function:${fn.functionName}:${alias.aliasName}`,
  scalableDimension: 'lambda:function:ProvisionedConcurrentExecutions',
});

// Scale based on utilization
target.scaleToTrackMetric('ProvisonedConcurrencyTracking', {
  targetValue: 0.7,
  predefinedMetric: applicationautoscaling.PredefinedMetric.LAMBDA_PROVISIONED_CONCURRENCY_UTILIZATION,
});
```

### Cost Monitoring

```bash
# Set up cost alerts
aws cloudwatch put-metric-alarm \
  --alarm-name WristAgent-MonthlyCost \
  --alarm-description "Alert when monthly costs exceed budget" \
  --metric-name EstimatedCharges \
  --namespace AWS/Billing \
  --statistic Maximum \
  --period 86400 \
  --threshold 100 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=Currency,Value=USD
```

## Backup and Recovery

### Backup Strategy

```bash
# Export CDK template
cd cdk
npx cdk synth > backup/template-$(date +%Y%m%d).json

# Backup SSM parameters
aws ssm get-parameter \
  --name "/wrist-agent/client-token" \
  --with-decryption \
  --query 'Parameter.Value' \
  --output text > backup/token-$(date +%Y%m%d).txt

# Backup CloudFormation stack
aws cloudformation get-template \
  --stack-name WristAgentStack \
  --query 'TemplateBody' > backup/stack-$(date +%Y%m%d).json
```

### Disaster Recovery

```bash
# Full stack recovery
# 1. Restore from backup
cd wrist-agent/cdk

# 2. Deploy to new region
export AWS_REGION=us-east-1
npx cdk bootstrap
npx cdk deploy

# 3. Restore token
aws ssm put-parameter \
  --name "/wrist-agent/client-token" \
  --value "$(cat backup/token-YYYYMMDD.txt)" \
  --type String

# 4. Update Apple Shortcut with new Function URL
```

## Rollback Procedures

### Immediate Rollback

```bash
# Rollback to previous version
aws lambda update-function-configuration \
  --function-name WristAgentStack-WristAgentHandler \
  --revision-id PREVIOUS_REVISION_ID

# Or destroy and redeploy previous version
git checkout v1.0.0
cd cdk
npx cdk deploy --require-approval never
```

### Gradual Rollback

Using Lambda aliases and weighted routing:

```typescript
// Shift traffic back to previous version
alias.addVersion(previousVersion, 1.0); // 100% to old
alias.addVersion(newVersion, 0.0);      // 0% to new
```

## Next Steps

- **[Monitor Your Deployment](./troubleshooting)** - Set up monitoring
- **[Secure Your System](./security)** - Review security best practices
- **[Optimize Performance](./examples)** - Learn advanced usage patterns
