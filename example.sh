#!/bin/bash

# Example script demonstrating route-switch functionality
# Note: This example uses the mock provider. For real API usage, create a config file with your API keys.

echo "=== Route-Switch Examples ==="
echo

echo "1. Optimizing a prompt for a specific model (using mock provider):"
echo "./route-switch --prompt \"Write a short story about a robot learning to cook\" --model \"gpt-4\" --provider mock --optimize-prompt"
echo
./route-switch --prompt "Write a short story about a robot learning to cook" --model "gpt-4" --provider mock --optimize-prompt
echo

echo "2. Finding the best model and optimized prompt combination (using mock provider):"
echo "./route-switch --prompt \"Explain quantum computing in simple terms\" --model \"gpt-4\" --provider mock --find-best-model"
echo
./route-switch --prompt "Explain quantum computing in simple terms" --model "gpt-4" --provider mock --find-best-model
echo

echo "3. Another example with model switching (using mock provider):"
echo "./route-switch --prompt \"Write a poem about the changing seasons\" --model \"gpt-4\" --provider mock --find-best-model"
echo
./route-switch --prompt "Write a poem about the changing seasons" --model "gpt-4" --provider mock --find-best-model
echo

echo "=== End of Examples ==="
echo
echo "To use real APIs, create a config file and use:"
echo "./route-switch --config config.yaml --prompt \"Your prompt\" --model \"gpt-4\" --provider gollm --find-best-model"
