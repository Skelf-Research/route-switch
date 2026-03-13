#!/bin/bash

# Example script demonstrating route-switch functionality

echo "=== Route-Switch Examples ==="
echo

echo "1. Optimizing a prompt for a specific model:"
echo "./route-switch --prompt \"Write a short story about a robot learning to cook\" --model \"gpt-4\" --optimize-prompt"
echo
./route-switch --prompt "Write a short story about a robot learning to cook" --model "gpt-4" --optimize-prompt
echo

echo "2. Finding the best model and optimized prompt combination:"
echo "./route-switch --prompt \"Explain quantum computing in simple terms\" --model \"gpt-4\" --find-best-model"
echo
./route-switch --prompt "Explain quantum computing in simple terms" --model "gpt-4" --find-best-model
echo

echo "3. Another example with model switching:"
echo "./route-switch --prompt \"Write a poem about the changing seasons\" --model \"gpt-4\" --find-best-model"
echo
./route-switch --prompt "Write a poem about the changing seasons" --model "gpt-4" --find-best-model
echo

echo "=== End of Examples ==="