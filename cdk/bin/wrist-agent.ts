#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import * as crypto from 'crypto';
import { WristAgentStack } from '../lib/wrist-agent-stack';

const app = new cdk.App();

// Get configuration from environment or use defaults
const region = process.env.AWS_REGION || 'us-west-2';
const modelId = process.env.BEDROCK_MODEL_ID || 'anthropic.claude-haiku-4-5-20251001-v1:0';
const clientTokenParamName = process.env.CLIENT_TOKEN_PARAM_NAME || '/wrist-agent/client-token';
const clientTokenValue = process.env.CLIENT_TOKEN || crypto.randomBytes(32).toString('base64');

new WristAgentStack(app, 'WristAgentStack', {
  env: {
    region: region,
  },
  description: 'Wrist Agent - Apple Watch to AWS Bedrock integration system',
  tags: {
    Project: 'WristAgent',
    Environment: process.env.DEPLOYMENT_ENVIRONMENT || 'dev',
  },
  config: {
    region: region,
    modelId: modelId,
    clientTokenParamName: clientTokenParamName,
    clientTokenValue: clientTokenValue,
  },
});
