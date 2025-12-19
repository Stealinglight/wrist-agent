---
sidebar_position: 5
---

# API Examples

This guide provides practical examples for using the Wrist Agent API across different modes and scenarios.

## Basic Request Structure

All requests follow this JSON structure:

```json
{
  "text": "User input text from voice capture",
  "mode": "note|reminder|event|research|deepthink",
  "maxTokens": 800,
  "thinkingTokens": 0
}
```

## Response Structure

All responses include these fields:

```json
{
  "markdown": "# Formatted content with markdown",
  "action": "note|reminder|event",
  "title": "Item title",
  "dueISO": "2025-01-16T14:00:00Z",
  "startISO": "2025-01-16T14:00:00Z",
  "endISO": "2025-01-16T15:00:00Z",
  "location": "Location string",
  "url": "https://example.com",
  "notes": "Additional notes",
  "tags": ["tag1", "tag2"]
}
```

## Mode Examples

### Note Mode

Create formatted notes from voice input.

**Request:**

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Met with Sarah today to discuss the Q1 marketing strategy. Key points: focus on social media campaigns, increase budget by 20%, target millennial audience",
    "mode": "note",
    "maxTokens": 800,
    "thinkingTokens": 0
  }'
```

**Response:**

```json
{
  "markdown": "# Q1 Marketing Strategy Meeting\n\n## Date\nToday\n\n## Attendees\n- Sarah\n- Me\n\n## Key Points\n\n1. **Focus Area**: Social media campaigns\n2. **Budget**: Increase by 20%\n3. **Target Audience**: Millennials\n\n## Next Steps\n- Review social media analytics\n- Prepare budget proposal\n- Research millennial engagement strategies",
  "action": "note",
  "title": "Q1 Marketing Strategy Meeting",
  "tags": ["meeting", "marketing", "strategy"]
}
```

### Reminder Mode

Create reminders with automatic date/time extraction.

**Request:**

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Remind me to call the dentist tomorrow at 2pm to schedule a cleaning",
    "mode": "reminder",
    "maxTokens": 400,
    "thinkingTokens": 0
  }'
```

**Response:**

```json
{
  "markdown": "# Call Dentist\n\nSchedule teeth cleaning appointment",
  "action": "reminder",
  "title": "Call dentist to schedule cleaning",
  "dueISO": "2025-01-20T14:00:00Z",
  "tags": ["health", "dentist"]
}
```

### Calendar Event Mode

Create calendar events with intelligent date/time parsing.

**Request:**

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Schedule team standup for next Monday at 10am for one hour, use the conference room zoom link",
    "mode": "event",
    "maxTokens": 600,
    "thinkingTokens": 0
  }'
```

**Response:**

```json
{
  "markdown": "# Team Standup\n\nRegular team sync meeting",
  "action": "event",
  "title": "Team Standup",
  "startISO": "2025-01-20T10:00:00Z",
  "endISO": "2025-01-20T11:00:00Z",
  "location": "Conference Room",
  "url": "https://zoom.us/j/your-meeting-id",
  "notes": "Regular team standup meeting",
  "tags": ["meeting", "team"]
}
```

### Research Mode

Get detailed research responses with structured information.

**Request:**

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "What are the main differences between REST and GraphQL APIs?",
    "mode": "research",
    "maxTokens": 1500,
    "thinkingTokens": 0
  }'
```

**Response:**

```json
{
  "markdown": "# REST vs GraphQL APIs\n\n## REST (Representational State Transfer)\n\n### Characteristics\n- Multiple endpoints for different resources\n- Fixed data structures in responses\n- HTTP methods (GET, POST, PUT, DELETE)\n- Over-fetching or under-fetching common\n\n### Pros\n- Simple and widely understood\n- Better caching with HTTP\n- Mature tooling and ecosystem\n\n### Cons\n- Multiple round trips for complex data\n- Version management challenges\n- Fixed response structures\n\n## GraphQL\n\n### Characteristics\n- Single endpoint for all queries\n- Client specifies exact data needs\n- Strongly typed schema\n- Nested queries in single request\n\n### Pros\n- Precise data fetching\n- Single request for complex data\n- Strong typing and introspection\n- Real-time updates with subscriptions\n\n### Cons\n- More complex to implement\n- Caching is more challenging\n- Steeper learning curve\n\n## When to Use Each\n\n**Use REST when:**\n- Simple CRUD operations\n- Standard HTTP caching needed\n- Team familiarity with REST\n\n**Use GraphQL when:**\n- Complex data relationships\n- Mobile apps with bandwidth concerns\n- Need flexible client queries",
  "action": "note",
  "title": "REST vs GraphQL APIs",
  "tags": ["research", "api", "technology"]
}
```

### Deep Think Mode

Enable extended reasoning for complex analysis.

**Request:**

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Should I buy or rent a house given current market conditions, interest rates, and my personal situation of being newly married with plans to start a family?",
    "mode": "deepthink",
    "maxTokens": 2000,
    "thinkingTokens": 10000
  }'
```

**Response:**

```json
{
  "markdown": "# Housing Decision: Buy vs Rent\n\n## Current Context\n- Newly married\n- Planning to start a family\n- Current market: high interest rates\n\n## Analysis\n\n### Buying Pros\n- Building equity over time\n- Stability for growing family\n- Tax benefits (mortgage interest deduction)\n- Fixed monthly costs (with fixed-rate mortgage)\n- Freedom to customize\n\n### Buying Cons\n- High upfront costs (down payment, closing costs)\n- Current elevated interest rates increase total cost\n- Less flexibility to relocate\n- Maintenance and repair responsibilities\n- Market risk if prices decline\n\n### Renting Pros\n- Lower upfront costs\n- Flexibility to move\n- No maintenance responsibilities\n- Can wait for better market conditions\n- Predictable monthly costs\n\n### Renting Cons\n- No equity building\n- Rent increases over time\n- Less stability for family\n- No tax benefits\n- Limited customization\n\n## Recommendation\n\n### Short-term (1-2 years)\nConsider renting while:\n- Building larger down payment\n- Establishing dual income stability\n- Monitoring interest rate trends\n- Identifying ideal neighborhoods for family\n\n### Long-term (3-5 years)\nLean toward buying when:\n- You have 20% down payment saved\n- Interest rates stabilize or decrease\n- You've identified where you want to raise children\n- Job stability is established\n\n## Key Factors to Consider\n\n1. **Financial**: Calculate total cost of ownership vs renting over 5 years\n2. **Lifestyle**: How important is location flexibility?\n3. **Timeline**: When do you plan to start your family?\n4. **Market**: Local housing market trends in your area\n5. **Risk tolerance**: Comfort with debt and market risk\n\n## Next Steps\n\n1. Calculate your budget and savings timeline\n2. Meet with financial advisor about mortgage pre-qualification\n3. Research neighborhoods with good schools\n4. Monitor interest rate trends\n5. Continue saving while renting\n\nThere's no universal right answer - it depends on your specific financial situation, timeline, and priorities.",
  "action": "note",
  "title": "Housing Decision Analysis",
  "tags": ["finance", "decision", "housing"]
}
```

## Advanced Usage

### Batch Processing

Process multiple inputs sequentially:

```bash
#!/bin/bash

FUNCTION_URL="https://your-function-url"
CLIENT_TOKEN="your-token"

# Array of tasks
declare -a tasks=(
  "note:Review meeting notes from standup"
  "reminder:Call mom on Sunday at 3pm"
  "event:Dentist appointment next Friday at 2pm"
)

# Process each task
for task in "${tasks[@]}"; do
  IFS=':' read -r mode text <<< "$task"
  
  curl -X POST "$FUNCTION_URL" \
    -H "Content-Type: application/json" \
    -H "X-Client-Token: $CLIENT_TOKEN" \
    -d "{
      \"text\": \"$text\",
      \"mode\": \"$mode\",
      \"maxTokens\": 800
    }"
  
  echo ""
done
```

### Error Handling

Handle API errors gracefully:

```bash
#!/bin/bash

response=$(curl -s -w "\n%{http_code}" -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Test input",
    "mode": "note"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "200" ]; then
  echo "Success: $body"
elif [ "$http_code" = "401" ]; then
  echo "Authentication failed. Check your token."
elif [ "$http_code" = "500" ]; then
  echo "Server error. Check CloudWatch logs."
else
  echo "Unexpected error: HTTP $http_code"
fi
```

### Testing Different Token Limits

Experiment with token limits for different use cases:

```bash
# Quick note - minimal tokens
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Quick note about coffee chat",
    "mode": "note",
    "maxTokens": 200
  }'

# Detailed note - more tokens
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Comprehensive meeting notes with action items",
    "mode": "note",
    "maxTokens": 1500
  }'

# Deep analysis - maximum tokens
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $CLIENT_TOKEN" \
  -d '{
    "text": "Complex business strategy analysis",
    "mode": "deepthink",
    "maxTokens": 4000,
    "thinkingTokens": 20000
  }'
```

## Next Steps

- **[Review Security Best Practices](./security)** - Protect your deployment
- **[Troubleshoot Issues](./troubleshooting)** - Resolve common problems
- **[Customize Your Shortcut](./apple-shortcut)** - Enhance your workflow
