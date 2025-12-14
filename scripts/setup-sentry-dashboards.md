# Sentry Dashboard Setup Guide for magda-api

## 🎯 Recommended Dashboards

### 1. **API Performance Dashboard**
Create a dashboard to monitor API performance:

**Widgets to add:**
- **Response Time (P95)**: `generation.generate` transaction duration
- **Request Volume**: Count of `generation.generate` transactions
- **Error Rate**: Failed `generation.generate` transactions
- **OpenAI API Performance**: `openai.api_call` span duration
- **Token Usage**: `openai.tokens.total` measurement

**Filters:**
- Environment: `production`
- Transaction: `generation.generate`

### 2. **Release Health Dashboard**
Track deployment health and regressions:

**Widgets to add:**
- **Crash-free Sessions**: Overall application health
- **Error Rate by Release**: Compare error rates across releases
- **Performance by Release**: Response time trends
- **Deployment Impact**: Error spikes around deployments

**Filters:**
- Environment: `production`
- Release: `magda-api@*`

### 3. **Business Metrics Dashboard**
Monitor business-specific metrics:

**Widgets to add:**
- **MCP Usage Rate**: `mcp_used` tag = true vs false
- **MCP Call Count**: `mcp.calls` measurement
- **Model Distribution**: `model` tag breakdown
- **Token Consumption**: `openai.tokens.*` measurements
- **Generation Success Rate**: `success` tag = true vs false

**Filters:**
- Environment: `production`
- Transaction: `generation.generate`

## 🚀 How to Create Dashboards

1. **Go to Sentry Dashboard**: https://sentry.io/organizations/[your-org]/dashboards/
2. **Click "Create Dashboard"**
3. **Add Widgets** using the specifications above
4. **Set Filters** for each widget
5. **Save Dashboard** with a descriptive name

## 📊 Key Metrics to Monitor

### Performance Metrics:
- `generation.duration` - Total generation time
- `openai.api_call` - OpenAI API response time
- `openai.tokens.total` - Token consumption
- `openai.tokens.reasoning` - Reasoning token usage (GPT-5)

### Business Metrics:
- `mcp_used` - Whether MCP server was used
- `mcp.calls` - Number of MCP tool calls
- `model` - Which OpenAI model was used
- `success` - Whether generation succeeded

### Error Tracking:
- `error_type` - Type of error (openai_api_error, json_parse_error, etc.)
- `openai_api_error` - OpenAI API failures
- `response_processing_error` - Response parsing failures

## 🔍 Useful Queries

### Find slow generations:
```
transaction:generation.generate
measurements.generation.duration:>5000
```

### Find MCP usage patterns:
```
transaction:generation.generate
tags.mcp_used:true
```

### Find token-heavy requests:
```
transaction:generation.generate
measurements.openai.tokens.total:>10000
```

### Find errors by type:
```
transaction:generation.generate
tags.success:false
```

## 📈 Alerts to Set Up

1. **High Error Rate**: >5% error rate for 5 minutes
2. **Slow Response Time**: P95 >10 seconds for 5 minutes
3. **High Token Usage**: Average tokens >20k for 10 minutes
4. **MCP Server Down**: MCP errors >10% for 5 minutes

## 🎵 Custom Tags Available

- `model`: OpenAI model used (gpt-5-mini, etc.)
- `mcp_enabled`: Whether MCP server is configured
- `mcp_used`: Whether MCP was actually used
- `success`: Whether generation succeeded
- `error_type`: Type of error if any
- `input_count`: Number of input compositions

## 📱 Mobile Dashboard

Create a mobile-friendly dashboard with:
- Current error rate
- Response time trend
- Active releases
- Recent errors

This gives you a quick health check on mobile!
