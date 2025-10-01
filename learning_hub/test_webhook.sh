#!/bin/bash

echo "🧪 Testing Chapa Webhook"
echo "========================"

# Simulate a successful payment webhook
WEBHOOK_URL="http://localhost:8080/api/webhooks/chapa"
TX_REF="learnhub-1234567890-abc123def" # Use the actual tx_ref from payment initiation

WEBHOOK_DATA='{
  "trx_ref": "'$TX_REF'",
  "ref_id": "CHAPA_TEST_REF_123",
  "status": "success"
}'

echo "Sending webhook to: $WEBHOOK_URL"
echo "Data: $WEBHOOK_DATA"

response=$(curl -s -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_DATA")

echo "Webhook Response: $response"