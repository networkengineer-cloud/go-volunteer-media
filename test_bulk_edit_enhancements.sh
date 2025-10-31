#!/bin/bash

# Bulk Edit Enhancements - Manual Testing Guide
# This script provides a structured approach to testing the new features

set -e

echo "=========================================="
echo "Bulk Edit Enhancements - Testing Guide"
echo "=========================================="
echo ""

# Check if backend and frontend are running
check_services() {
    echo "🔍 Checking if services are running..."
    
    # Check backend
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "✅ Backend is running on port 8080"
    else
        echo "❌ Backend is NOT running. Start it with: make dev-backend"
        exit 1
    fi
    
    # Check frontend
    if curl -s http://localhost:5173 > /dev/null; then
        echo "✅ Frontend is running on port 5173"
    else
        echo "❌ Frontend is NOT running. Start it with: cd frontend && npm run dev"
        exit 1
    fi
    
    echo ""
}

# Test 1: Individual Status Change
test_status_change() {
    echo "=========================================="
    echo "TEST 1: Individual Status Change"
    echo "=========================================="
    echo ""
    echo "Prerequisites:"
    echo "- You must be logged in as an admin"
    echo "- Navigate to http://localhost:5173/admin/animals"
    echo ""
    echo "Test Steps:"
    echo "1. Locate any animal in the table"
    echo "2. Find the 'Status' column (should show a dropdown)"
    echo "3. Click the status dropdown"
    echo "4. Select a different status (Available/Adopted/Fostered)"
    echo "5. Observe: Row should briefly show reduced opacity"
    echo "6. Verify: Status is updated immediately"
    echo ""
    echo "Expected Result:"
    echo "✅ Status changes immediately without page reload"
    echo "✅ Row shows visual feedback during update"
    echo "✅ Table reflects new status"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 2: Individual Group Change
test_group_change() {
    echo "=========================================="
    echo "TEST 2: Individual Group Change"
    echo "=========================================="
    echo ""
    echo "Prerequisites:"
    echo "- You must be logged in as an admin"
    echo "- Navigate to http://localhost:5173/admin/animals"
    echo ""
    echo "Test Steps:"
    echo "1. Locate any animal in the table"
    echo "2. Find the 'Group' column (should show a dropdown)"
    echo "3. Click the group dropdown"
    echo "4. Select a different group from the list"
    echo "5. Observe: Row should briefly show reduced opacity"
    echo "6. Verify: Group is updated immediately"
    echo ""
    echo "Expected Result:"
    echo "✅ Group changes immediately without page reload"
    echo "✅ Row shows visual feedback during update"
    echo "✅ Table reflects new group"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 3: Export Comments - All Animals
test_export_all_comments() {
    echo "=========================================="
    echo "TEST 3: Export All Animal Comments"
    echo "=========================================="
    echo ""
    echo "Prerequisites:"
    echo "- You must be logged in as an admin"
    echo "- Navigate to http://localhost:5173/admin/animals"
    echo "- Ensure some animals have comments"
    echo ""
    echo "Test Steps:"
    echo "1. Click the 'Export Comments' button in the page header"
    echo "2. Verify: A CSV file 'animal-comments.csv' downloads"
    echo "3. Open the CSV file in Excel or a text editor"
    echo "4. Verify columns exist:"
    echo "   - comment_id, animal_id, animal_name, animal_species"
    echo "   - animal_breed, animal_status, group_id, group_name"
    echo "   - comment_content, comment_author, comment_tags"
    echo "   - created_at, updated_at"
    echo "5. Verify data is correctly populated"
    echo ""
    echo "Expected Result:"
    echo "✅ CSV file downloads automatically"
    echo "✅ All columns are present"
    echo "✅ Animal data is included with comments"
    echo "✅ Timestamps are in ISO 8601 format"
    echo "✅ Tags are semicolon-separated"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 4: Export Comments - Filtered by Group
test_export_filtered_comments() {
    echo "=========================================="
    echo "TEST 4: Export Filtered Animal Comments"
    echo "=========================================="
    echo ""
    echo "Prerequisites:"
    echo "- You must be logged in as an admin"
    echo "- Navigate to http://localhost:5173/admin/animals"
    echo "- Multiple groups exist with animals and comments"
    echo ""
    echo "Test Steps:"
    echo "1. Select a specific group from 'Filter by Group' dropdown"
    echo "2. Click the 'Export Comments' button"
    echo "3. Verify: A CSV file 'animal-comments.csv' downloads"
    echo "4. Open the CSV file"
    echo "5. Verify: Only comments for animals in selected group"
    echo "6. Check the 'group_name' column matches selected group"
    echo ""
    echo "Expected Result:"
    echo "✅ CSV contains only filtered group's comments"
    echo "✅ Group names match the filter selection"
    echo "✅ No comments from other groups included"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 5: Multiple Quick Changes
test_multiple_changes() {
    echo "=========================================="
    echo "TEST 5: Multiple Quick Status Changes"
    echo "=========================================="
    echo ""
    echo "Prerequisites:"
    echo "- You must be logged in as an admin"
    echo "- Navigate to http://localhost:5173/admin/animals"
    echo ""
    echo "Test Steps:"
    echo "1. Change status of first animal"
    echo "2. Immediately change status of second animal"
    echo "3. Change group of third animal"
    echo "4. Verify: All changes are saved correctly"
    echo "5. Reload page and verify changes persisted"
    echo ""
    echo "Expected Result:"
    echo "✅ Multiple quick changes are handled correctly"
    echo "✅ No race conditions or lost updates"
    echo "✅ Changes persist after page reload"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 6: Bulk Actions Still Work
test_bulk_actions() {
    echo "=========================================="
    echo "TEST 6: Verify Bulk Actions Still Work"
    echo "=========================================="
    echo ""
    echo "Prerequisites:"
    echo "- You must be logged in as an admin"
    echo "- Navigate to http://localhost:5173/admin/animals"
    echo ""
    echo "Test Steps:"
    echo "1. Select multiple animals using checkboxes"
    echo "2. Choose 'Change Status' from bulk action dropdown"
    echo "3. Select a status and click 'Apply'"
    echo "4. Verify: All selected animals update"
    echo "5. Try 'Move to Group' bulk action"
    echo "6. Verify: All selected animals move to new group"
    echo ""
    echo "Expected Result:"
    echo "✅ Bulk actions work as before"
    echo "✅ Multiple animals update simultaneously"
    echo "✅ No interference with individual dropdowns"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 7: Edge Cases
test_edge_cases() {
    echo "=========================================="
    echo "TEST 7: Edge Cases"
    echo "=========================================="
    echo ""
    echo "Test Scenarios:"
    echo ""
    echo "A. Export comments when no comments exist:"
    echo "   - Should download empty CSV with headers only"
    echo ""
    echo "B. Change status while another update is in progress:"
    echo "   - Dropdown should be disabled during update"
    echo ""
    echo "C. Export with no group filter:"
    echo "   - Should export all comments from all groups"
    echo ""
    echo "D. Change animal to same status/group:"
    echo "   - Should still make API call (idempotent)"
    echo ""
    echo "Expected Results:"
    echo "✅ All edge cases handled gracefully"
    echo "✅ No errors or crashes"
    echo "✅ Appropriate user feedback"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# Test 8: Visual Inspection
test_visual_inspection() {
    echo "=========================================="
    echo "TEST 8: Visual Inspection"
    echo "=========================================="
    echo ""
    echo "Visual Checks:"
    echo ""
    echo "1. Dropdowns style consistently:"
    echo "   - Border color changes on hover"
    echo "   - Focus state shows brand color"
    echo "   - Disabled state shows reduced opacity"
    echo ""
    echo "2. Page header buttons:"
    echo "   - Three buttons: Export Comments | Export Animals | Import CSV"
    echo "   - Proper spacing and alignment"
    echo ""
    echo "3. Table layout:"
    echo "   - Dropdowns fit well in columns"
    echo "   - No overflow or alignment issues"
    echo "   - Responsive on mobile devices"
    echo ""
    echo "4. Loading states:"
    echo "   - Row opacity reduces during update"
    echo "   - Returns to normal after completion"
    echo ""
    echo "Expected Results:"
    echo "✅ Consistent styling across all elements"
    echo "✅ No layout issues or overlaps"
    echo "✅ Good user experience"
    echo ""
    read -p "Press Enter when test is complete or Ctrl+C to exit..."
    echo ""
}

# API Testing with curl
test_api_endpoints() {
    echo "=========================================="
    echo "TEST 9: API Endpoints (curl tests)"
    echo "=========================================="
    echo ""
    echo "Note: You'll need an admin JWT token for these tests"
    echo ""
    read -p "Enter your JWT token (or press Enter to skip): " JWT_TOKEN
    echo ""
    
    if [ -z "$JWT_TOKEN" ]; then
        echo "⏭️  Skipping API tests"
        return
    fi
    
    echo "Testing GET /api/admin/animals/export-comments-csv"
    echo "---"
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        http://localhost:8080/api/admin/animals/export-comments-csv)
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)
    
    if [ "$HTTP_CODE" == "200" ]; then
        echo "✅ Export comments endpoint returns 200"
    else
        echo "❌ Export comments endpoint returned: $HTTP_CODE"
    fi
    echo ""
    
    echo "Testing PUT /api/admin/animals/1 (update animal)"
    echo "---"
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X PUT \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"status":"available"}' \
        http://localhost:8080/api/admin/animals/1)
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)
    
    if [ "$HTTP_CODE" == "200" ]; then
        echo "✅ Update animal endpoint returns 200"
    else
        echo "❌ Update animal endpoint returned: $HTTP_CODE"
    fi
    echo ""
    
    read -p "Press Enter to continue..."
    echo ""
}

# Main test runner
main() {
    echo "This script guides you through testing the bulk edit enhancements."
    echo "Make sure both backend and frontend are running before starting."
    echo ""
    read -p "Press Enter to start testing..."
    echo ""
    
    check_services
    
    test_status_change
    test_group_change
    test_export_all_comments
    test_export_filtered_comments
    test_multiple_changes
    test_bulk_actions
    test_edge_cases
    test_visual_inspection
    test_api_endpoints
    
    echo "=========================================="
    echo "✅ All Tests Complete!"
    echo "=========================================="
    echo ""
    echo "Summary:"
    echo "- Individual status dropdowns tested"
    echo "- Individual group dropdowns tested"
    echo "- Comment export functionality tested"
    echo "- Filtered export tested"
    echo "- Bulk actions verified"
    echo "- Edge cases covered"
    echo "- Visual inspection complete"
    echo ""
    echo "If all tests passed, the features are working correctly!"
    echo ""
}

# Run main function
main
