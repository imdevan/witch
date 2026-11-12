# Colors
RED='\033[0;31m'
NC='\033[0m'

# Print text in red
red() {
  printf "${RED}%s${NC}\n" "$*"
}
