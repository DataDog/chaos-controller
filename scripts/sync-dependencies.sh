#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default file paths
LOCAL_GO_MOD="./go.mod"
REFERENCE_GO_MOD="./reference.go.mod"
REFERENCE_REPO=""
DRY_RUN=false
VERBOSE=false
CLEANUP_DOWNLOADED=false

# Usage function
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Sync Go version and shared dependencies between local go.mod and reference.go.mod"
    echo ""
    echo "Options:"
    echo "  -l, --local FILE      Path to local go.mod file (default: ./go.mod)"
    echo "  -d, --reference FILE  Path to reference.go.mod file (default: ./reference.go.mod)"
    echo "  -r, --repo REPO       GitHub repository to download go.mod from"
    echo "  -n, --dry-run         Show what would be changed without making changes"
    echo "  -v, --verbose         Enable verbose output"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Repository formats (using GitHub CLI):"
    echo "  owner/repo                            # go.mod from main branch"
    echo "  owner/repo:branch                     # go.mod from specific branch"
    echo "  owner/repo:branch:path/to/go.mod      # specific file and branch"
    echo "  https://github.com/owner/repo/blob/branch/go.mod"
    echo "  https://raw.githubusercontent.com/owner/repo/branch/go.mod"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Sync using default files"
    echo "  $0 --dry-run                         # Preview changes without applying"
    echo "  $0 -d /path/to/reference.go.mod      # Use local reference file"
    echo "  $0 -r owner/reference-repo            # Download from GitHub repo (main branch)"
    echo "  $0 -r owner/reference-repo:develop    # Download from specific branch"
    echo "  $0 -r owner/reference-repo:main:go.mod # Download specific file"
    echo ""
    echo "Note:"
    echo "  - GitHub CLI (gh) must be installed and authenticated for repository access"
    echo "  - Install GitHub CLI: https://cli.github.com/ or 'brew install gh'"
    echo "  - Run 'gh auth login' to authenticate if needed"
    echo "  - Private repositories are supported through GitHub CLI authentication"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -l|--local)
            LOCAL_GO_MOD="$2"
            shift 2
            ;;
        -d|--reference)
            REFERENCE_GO_MOD="$2"
            shift 2
            ;;
        -r|--repo)
            REFERENCE_REPO="$2"
            shift 2
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${NC}[VERBOSE] $1"
    fi
}

# Download reference go.mod from repository using GitHub CLI
download_reference_github() {
    local repo="$1"
    local file_path="$2"
    local ref="${3:-main}"

    log_info "Downloading go.mod from GitHub repo: $repo (ref: $ref)"
    log_verbose "File path: $file_path"

    # Check if gh CLI is available
    if ! command -v gh >/dev/null 2>&1; then
        log_error "GitHub CLI (gh) is not installed or not in PATH"
        log_info "Please install it from: https://cli.github.com/"
        cleanup_temp_files
        exit 1
    fi

    # Check if user is authenticated
    if ! gh auth status >/dev/null 2>&1; then
        log_error "GitHub CLI is not authenticated"
        log_info "Please run: gh auth login"
        cleanup_temp_files
        exit 1
    fi

    # Create a temporary file
    REFERENCE_GO_MOD=$(mktemp -t reference-go.mod.XXXXXX)
    CLEANUP_DOWNLOADED=true
    log_verbose "Temporary file: $REFERENCE_GO_MOD"

    # Download file using gh CLI
    if gh api "repos/$repo/contents/$file_path?ref=$ref" --jq '.content' | base64 -d > "$REFERENCE_GO_MOD"; then
        # Verify the downloaded file looks like a go.mod
        if grep -q "^module " "$REFERENCE_GO_MOD" 2>/dev/null; then
            log_success "Successfully downloaded go.mod from $repo"
            return 0
        else
            log_error "Downloaded file doesn't appear to be a valid go.mod file"
        fi
    else
        log_error "Failed to download go.mod from $repo"
        log_info "Make sure the repository exists and you have access to it"
    fi

    cleanup_temp_files
    exit 1
}

# Parse GitHub repository information from various URL formats
parse_github_repo() {
    local input="$1"
    local repo=""
    local file_path=""
    local ref=""

    if [[ "$input" == *"github.com"* ]]; then
        # Handle GitHub URLs
        if [[ "$input" == *"/blob/"* ]]; then
            # Format: https://github.com/owner/repo/blob/branch/path/to/file
            repo=$(echo "$input" | sed -E 's|.*github\.com/([^/]+/[^/]+)/blob/.*|\1|')
            ref=$(echo "$input" | sed -E 's|.*github\.com/[^/]+/[^/]+/blob/([^/]+)/.*|\1|')
            file_path=$(echo "$input" | sed -E 's|.*github\.com/[^/]+/[^/]+/blob/[^/]+/(.*)|\1|')
        elif [[ "$input" == *"/raw/"* ]]; then
            # Format: https://raw.githubusercontent.com/owner/repo/branch/path/to/file
            repo=$(echo "$input" | sed -E 's|.*raw\.githubusercontent\.com/([^/]+/[^/]+)/.*|\1|')
            ref=$(echo "$input" | sed -E 's|.*raw\.githubusercontent\.com/[^/]+/[^/]+/([^/]+)/.*|\1|')
            file_path=$(echo "$input" | sed -E 's|.*raw\.githubusercontent\.com/[^/]+/[^/]+/[^/]+/(.*)|\1|')
        else
            log_error "Unsupported GitHub URL format: $input"
            log_info "Supported formats:"
            log_info "  - https://github.com/owner/repo/blob/branch/go.mod"
            log_info "  - https://raw.githubusercontent.com/owner/repo/branch/go.mod"
            log_info "  - owner/repo (assumes go.mod at root, main branch)"
            log_info "  - owner/repo:branch"
            log_info "  - owner/repo:branch:path/to/go.mod"
            exit 1
        fi
    else
        # Handle shorthand formats
        if [[ "$input" == *":"* ]]; then
            # Format: owner/repo:branch or owner/repo:branch:path
            repo=$(echo "$input" | cut -d':' -f1)
            local remaining=$(echo "$input" | cut -d':' -f2-)

            if [[ "$remaining" == *":"* ]]; then
                # Format: owner/repo:branch:path
                ref=$(echo "$remaining" | cut -d':' -f1)
                file_path=$(echo "$remaining" | cut -d':' -f2-)
            else
                # Format: owner/repo:branch
                ref="$remaining"
                file_path="go.mod"
            fi
        else
            # Format: owner/repo (assume main branch, go.mod at root)
            repo="$input"
            ref="main"
            file_path="go.mod"
        fi
    fi

    echo "$repo:$file_path:$ref"
}

# Download reference go.mod from various sources
download_reference() {
    local input="$1"

    if [[ "$input" == http* ]]; then
        # Full URL - use URL download
        download_reference_url "$input"
    elif [[ "$input" == *"github.com"* ]] || [[ "$input" =~ ^[^/]+/[^/]+(:.*)?$ ]]; then
        # GitHub repository format
        local parsed=$(parse_github_repo "$input")
        local repo=$(echo "$parsed" | cut -d':' -f1)
        local file_path=$(echo "$parsed" | cut -d':' -f2)
        local ref=$(echo "$parsed" | cut -d':' -f3)

        download_reference_github "$repo" "$file_path" "$ref"
    else
        # Fallback to URL download for other sources
        download_reference_url "$input"
    fi
}

# Fallback URL download for non-GitHub sources
download_reference_url() {
    local url="$1"
    log_info "Downloading reference go.mod from: $url"

    # Create a temporary file
    REFERENCE_GO_MOD=$(mktemp -t reference-go.mod.XXXXXX)
    CLEANUP_DOWNLOADED=true
    log_verbose "Temporary file: $REFERENCE_GO_MOD"

    # Try curl first, then wget
    if command -v curl >/dev/null 2>&1; then
        log_verbose "Using curl to download..."
        if curl -fsSL "$url" -o "$REFERENCE_GO_MOD"; then
            if grep -q "^module " "$REFERENCE_GO_MOD" 2>/dev/null; then
                log_success "Successfully downloaded reference go.mod"
                return 0
            else
                log_error "Downloaded file doesn't appear to be a valid go.mod file"
            fi
        else
            log_error "Failed to download with curl"
        fi
    elif command -v wget >/dev/null 2>&1; then
        log_verbose "Using wget to download..."
        if wget -q "$url" -O "$REFERENCE_GO_MOD"; then
            if grep -q "^module " "$REFERENCE_GO_MOD" 2>/dev/null; then
                log_success "Successfully downloaded reference go.mod"
                return 0
            else
                log_error "Downloaded file doesn't appear to be a valid go.mod file"
            fi
        else
            log_error "Failed to download with wget"
        fi
    else
        log_error "Neither curl nor wget is available for downloading"
    fi

    log_error "Failed to download reference go.mod from $url"
    cleanup_temp_files
    exit 1
}

# Cleanup temporary files
cleanup_temp_files() {
    if [[ "$CLEANUP_DOWNLOADED" == "true" && -f "$REFERENCE_GO_MOD" ]]; then
        log_verbose "Cleaning up temporary file: $REFERENCE_GO_MOD"
        rm -f "$REFERENCE_GO_MOD"
    fi
}

# Trap to ensure cleanup on exit
trap cleanup_temp_files EXIT

# Check if files exist
check_files() {
    if [[ ! -f "$LOCAL_GO_MOD" ]]; then
        log_error "Local go.mod file not found: $LOCAL_GO_MOD"
        exit 1
    fi

    # Handle reference go.mod - either download from repository or check local file
    if [[ -n "$REFERENCE_REPO" ]]; then
        download_reference "$REFERENCE_REPO"
    elif [[ ! -f "$REFERENCE_GO_MOD" ]]; then
        log_error "reference.go.mod file not found: $REFERENCE_GO_MOD"
        log_info "Use -d to specify the path or -r to download from GitHub repository"
        exit 1
    fi

    log_verbose "Using local go.mod: $LOCAL_GO_MOD"
    log_verbose "Using reference go.mod: $REFERENCE_GO_MOD"
}

# Extract dependencies from a go.mod file
# Excludes indirect dependencies and comments
extract_dependencies() {
    local go_mod_file="$1"
    log_verbose "Extracting dependencies from $go_mod_file"

    # Use awk to parse dependencies, excluding indirect ones and handling multi-line require blocks
    awk '
    /^require \(/ { in_require_block = 1; next }
    /^\)/ && in_require_block { in_require_block = 0; next }
    /^require / && !in_require_block {
        # Handle single line require
        sub(/^require /, "")
        if ($0 !~ /\/\/ indirect/ && $0 !~ /^\/\//) {
            print $1 " " $2
        }
        next
    }
    in_require_block && !/^\/\// && !/\/\/ indirect/ && NF >= 2 {
        # Handle require block entries, skip comments and indirect
        print $1 " " $2
    }
    ' "$go_mod_file" | sed 's/[[:space:]]*$//' | sort
}

# Find shared dependencies between two dependency lists
find_shared_dependencies() {
    local local_deps="$1"
    local reference_deps="$2"

    log_verbose "Finding shared dependencies..."

    # Extract package names only and find intersection
    local local_packages=$(echo "$local_deps" | cut -d' ' -f1 | sort)
    local reference_packages=$(echo "$reference_deps" | cut -d' ' -f1 | sort)

    # Find shared package names
    comm -12 <(echo "$local_packages") <(echo "$reference_packages")
}

# Sync Go version between the two go.mod files
sync_go_version() {
    log_verbose "Checking Go version synchronization..."

    # Extract Go versions from both files
    local local_go_version=$(grep "^go " "$LOCAL_GO_MOD" | awk '{print $2}')
    local reference_go_version=$(grep "^go " "$REFERENCE_GO_MOD" | awk '{print $2}')

    log_verbose "Local Go version: $local_go_version"
    log_verbose "Reference Go version: $reference_go_version"

    if [[ -z "$local_go_version" ]]; then
        log_warn "Could not find Go version in local go.mod"
        return 0
    fi

    if [[ -z "$reference_go_version" ]]; then
        log_warn "Could not find Go version in reference go.mod"
        return 0
    fi

    if [[ "$local_go_version" != "$reference_go_version" ]]; then
        echo ""
        log_info "Go version difference detected:"
        echo -e "  Local:     ${RED}$local_go_version${NC}"
        echo -e "  Reference: ${GREEN}$reference_go_version${NC}"
        echo ""

        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Dry run mode - would update Go version from $local_go_version to $reference_go_version"
        else
            echo -n "Do you want to update the local Go version to match reference? [y/N]: "
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                # Update Go version
                sed -i.bak "s|^go $local_go_version|go $reference_go_version|g" "$LOCAL_GO_MOD"
                rm -f "${LOCAL_GO_MOD}.bak"
                log_success "Updated Go version from $local_go_version to $reference_go_version"
            else
                log_info "Go version update skipped"
            fi
        fi
    else
        log_verbose "✓ Go versions match: $local_go_version"
    fi
    echo ""
}

# Compare versions of shared dependencies
compare_shared_dependencies() {
    local local_deps="$1"
    local reference_deps="$2"
    local shared_packages="$3"

    local differences_found=false
    local updates_needed=()

    log_info "Comparing shared dependency versions..."
    echo ""

    while IFS= read -r package; do
        [[ -z "$package" ]] && continue

        local local_version=$(echo "$local_deps" | grep "^$package " | cut -d' ' -f2)
        local reference_version=$(echo "$reference_deps" | grep "^$package " | cut -d' ' -f2)

        if [[ "$local_version" != "$reference_version" ]]; then
            differences_found=true
            echo -e "${YELLOW}📦 $package${NC}"
            echo -e "  Local:     ${RED}$local_version${NC}"
            echo -e "  Reference: ${GREEN}$reference_version${NC}"

            # Store update information
            updates_needed+=("$package $reference_version")
            echo ""
        else
            log_verbose "✓ $package versions match: $local_version"
        fi
    done <<< "$shared_packages"

    if [[ "$differences_found" == "false" ]]; then
        log_success "All shared dependencies are already synchronized! 🎉"
        return 0
    fi

    echo -e "${BLUE}Found ${#updates_needed[@]} dependencies that need updating${NC}"
    echo ""

    # Show what would be updated
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Dry run mode - would update the following:"
        for update in "${updates_needed[@]}"; do
            echo "  - $update"
        done
        return 0
    fi

    # Ask for confirmation
    echo -n "Do you want to update the local go.mod with reference versions? [y/N]: "
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        log_info "Update cancelled by user"
        return 0
    fi

    # Perform updates
    log_info "Updating dependencies..."
    local backup_file="${LOCAL_GO_MOD}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$LOCAL_GO_MOD" "$backup_file"
    log_info "Created backup: $backup_file"

    for update in "${updates_needed[@]}"; do
        local package_name=$(echo "$update" | cut -d' ' -f1)
        local new_version=$(echo "$update" | cut -d' ' -f2)

        log_verbose "Updating $package_name to $new_version"

        # Get the current version from the local go.mod (excluding indirect dependencies)
        local current_version=$(grep -E "^[[:space:]]*${package_name}[[:space:]]+" "$LOCAL_GO_MOD" | grep -v "// indirect" | awk '{print $2}' | head -1)

        if [[ -n "$current_version" ]]; then
            # Use a different delimiter for sed to avoid issues with forward slashes
            # Match tabs and spaces explicitly, preserve exact whitespace
            sed -i.bak "s|${package_name}[ \t][ \t]*${current_version}|${package_name} ${new_version}|g" "$LOCAL_GO_MOD"

            # Remove backup file
            rm -f "${LOCAL_GO_MOD}.bak"

            log_verbose "Updated $package_name from $current_version to $new_version"
        else
            log_verbose "Could not find $package_name in go.mod"
        fi
    done

    log_success "Dependencies updated successfully!"
    log_info "Run 'go mod tidy' to clean up the module file"

    return 0
}

# Main function
main() {
    log_info "🔄 Starting dependency synchronization..."
    echo ""

    check_files

    # Extract dependencies from both files
    log_verbose "Extracting dependencies..."
    local local_deps
    local reference_deps
    local_deps=$(extract_dependencies "$LOCAL_GO_MOD")
    reference_deps=$(extract_dependencies "$REFERENCE_GO_MOD")

    log_verbose "Found $(echo "$local_deps" | wc -l) dependencies in local go.mod"
    log_verbose "Found $(echo "$reference_deps" | wc -l) dependencies in reference go.mod"

    # Find shared dependencies
    local shared_packages
    shared_packages=$(find_shared_dependencies "$local_deps" "$reference_deps")

    local shared_count=$(echo "$shared_packages" | grep -c . || true)
    log_info "Found $shared_count shared dependencies"

    if [[ $shared_count -eq 0 ]]; then
        log_warn "No shared dependencies found between the two go.mod files"
        return 0
    fi

    # Compare and sync Go version first
    sync_go_version

    # Compare versions and potentially update
    compare_shared_dependencies "$local_deps" "$reference_deps" "$shared_packages"
}

# Run main function
main "$@"