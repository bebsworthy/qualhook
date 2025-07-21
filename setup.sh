#!/bin/bash

# Setup script for Spec Workflow
# This script configures the spec workflow for different AI assistants

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Reference text to add to CLAUDE.md or GEMINI.md
REFERENCE_TEXT="# MANDATORY: Spec Workflow

**⚠️ IMPORTANT: You MUST read \`.spec/WORKFLOW.md\` before starting any feature development.**

The spec workflow is mandatory for all feature development. It ensures systematic, high-quality implementation through defined phases:
1. Architecture Analysis
2. Feature Creation  
3. Research
4. Requirements
5. Design
6. Tasks
7. Implementation

See \`.spec/WORKFLOW.md\` for complete details.

---
"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to add reference to AI configuration file
add_reference_to_file() {
    local file="$1"
    local ai_name="$2"
    
    if [ -f "$PROJECT_ROOT/$file" ]; then
        # Check if reference already exists
        if grep -q "MANDATORY: Spec Workflow" "$PROJECT_ROOT/$file" 2>/dev/null; then
            print_info "$file already contains spec workflow reference"
        else
            print_info "Adding spec workflow reference to existing $file"
            # Create temporary file with reference at top
            echo "$REFERENCE_TEXT" > "$PROJECT_ROOT/$file.tmp"
            cat "$PROJECT_ROOT/$file" >> "$PROJECT_ROOT/$file.tmp"
            mv "$PROJECT_ROOT/$file.tmp" "$PROJECT_ROOT/$file"
            print_success "Added spec workflow reference to $file"
        fi
    else
        print_info "Creating $file with spec workflow reference"
        echo "$REFERENCE_TEXT" > "$PROJECT_ROOT/$file"
        print_success "Created $file with spec workflow reference"
    fi
}

# Function to setup Claude
setup_claude() {
    print_info "Setting up Claude configuration..."
    
    # Add reference to CLAUDE.md
    add_reference_to_file "CLAUDE.md" "Claude"
    
    # Create .claude/commands/spec directory
    mkdir -p "$PROJECT_ROOT/.claude/commands/spec"
    
    # Copy spec commands
    print_info "Copying spec commands to .claude/commands/spec/"
    cp -r "$SCRIPT_DIR/commands/spec/"* "$PROJECT_ROOT/.claude/commands/spec/"
    
    print_success "Claude setup complete"
}

# Function to setup Gemini
setup_gemini() {
    print_info "Setting up Gemini configuration..."
    
    # Add reference to GEMINI.md
    add_reference_to_file "GEMINI.md" "Gemini"
    
    # Create prompts directory
    mkdir -p "$PROJECT_ROOT/prompts"
    
    # Copy spec commands to prompts (Gemini doesn't support / commands)
    print_info "Copying spec commands to prompts/ directory"
    
    # Copy each command file with a descriptive name for Gemini
    for file in "$SCRIPT_DIR/commands/spec/"*.md; do
        if [ -f "$file" ]; then
            basename=$(basename "$file" .md)
            # Create Gemini-friendly filename
            gemini_name="spec_workflow_${basename}.md"
            cp "$file" "$PROJECT_ROOT/prompts/$gemini_name"
        fi
    done
    
    # Also copy the main WORKFLOW.md for reference
    cp "$SCRIPT_DIR/WORKFLOW.md" "$PROJECT_ROOT/prompts/spec_workflow_main.md"
    
    print_success "Gemini setup complete"
}

# Function to setup documentation directory
setup_documentation() {
    print_info "Setting up documentation directory..."
    mkdir -p "$PROJECT_ROOT/documentation/specs"
    print_success "Created documentation/specs directory"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [claude|gemini|all]"
    echo ""
    echo "Options:"
    echo "  claude    Setup for Claude AI"
    echo "  gemini    Setup for Gemini AI"
    echo "  all       Setup for all supported AIs"
    echo ""
    echo "If no option is provided, the script will run interactively."
}

# Main execution
main() {
    echo "======================================="
    echo "   Spec Workflow Setup Script"
    echo "======================================="
    echo ""
    
    # Check if an argument was provided
    if [ $# -eq 0 ]; then
        # Interactive mode
        echo "Which AI assistant are you using?"
        echo "1) Claude"
        echo "2) Gemini"
        echo "3) All"
        echo ""
        read -p "Enter your choice (1-3): " choice
        
        case $choice in
            1)
                AI_CHOICE="claude"
                ;;
            2)
                AI_CHOICE="gemini"
                ;;
            3)
                AI_CHOICE="all"
                ;;
            *)
                print_error "Invalid choice"
                exit 1
                ;;
        esac
    else
        # Command line argument mode
        AI_CHOICE="$1"
    fi
    
    # Validate choice
    case $AI_CHOICE in
        claude)
            setup_documentation
            setup_claude
            ;;
        gemini)
            setup_documentation
            setup_gemini
            ;;
        all)
            setup_documentation
            setup_claude
            setup_gemini
            ;;
        -h|--help|help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Invalid option: $AI_CHOICE"
            show_usage
            exit 1
            ;;
    esac
    
    echo ""
    echo "======================================="
    print_success "Setup completed successfully!"
    echo ""
    print_info "Next steps:"
    echo "  1. Read .spec/WORKFLOW.md to understand the spec workflow"
    echo "  2. Run /spec:architecture to analyze your codebase (one-time)"
    echo "  3. Start creating features with /spec:1_create"
    echo ""
}

# Run main function with all arguments
main "$@"