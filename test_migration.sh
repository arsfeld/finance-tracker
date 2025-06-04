#!/bin/bash

echo "Testing WalletMind database migration..."

# Check if Supabase is running
echo "1. Checking Supabase status..."
supabase status

# Run the migration
echo -e "\n2. Running migration..."
supabase db reset --debug

echo -e "\n3. Migration complete! You can now:"
echo "   - Run 'just web-dev' to start the web server"
echo "   - Visit http://localhost:8080/register to create an account"
echo "   - Check the logs for detailed authentication information"