#!/bin/bash

# Script to set up Hookit database schema on Supabase
# Run this script to create all necessary tables

# You can run this SQL in your Supabase SQL Editor:
# Go to https://app.supabase.com/project/YOUR_PROJECT_ID/sql/new
# Copy and paste the contents of schema.sql

echo "Database schema for Hookit is ready in schema.sql"
echo "Please run the SQL commands in your Supabase SQL Editor:"
echo "1. Go to https://app.supabase.com"
echo "2. Open your project"
echo "3. Go to SQL Editor"
echo "4. Copy and paste the contents of schema.sql"
echo "5. Click 'Run'"

cat schema.sql
