#!/bin/bash

# Exit on first error
set -e

# Import utils
. scripts/utils.sh

# Check for required tools
check_requirements() {
  local missing_requirements=false

  # Check for peer
  if ! command -v peer &> /dev/null; then
    echo -e "${RED}Error: 'peer' command not found. Please check your Fabric installation.${NC}"
    echo "Make sure \$PATH includes the Fabric bin directory."
    missing_requirements=true
  fi

  # Check for fabric-ca-client
  if ! command -v fabric-ca-client &> /dev/null; then
    echo -e "${RED}Error: 'fabric-ca-client' command not found. Please check your Fabric installation.${NC}"
    echo "Make sure \$PATH includes the Fabric bin directory."
    missing_requirements=true
  fi

  # Check for bc (used for calculations)
  if ! command -v bc &> /dev/null; then
    echo -e "${RED}Error: 'bc' command not found. Please install bc (basic calculator).${NC}"
    echo "For MacOS: brew install bc"
    echo "For Ubuntu: apt-get install bc"
    missing_requirements=true
  fi

  if [ "$missing_requirements" = true ]; then
    echo -e "${RED}Missing required tools. Please install them and try again.${NC}"
    exit 1
  fi
  
  echo "All required tools are available."
}

# Set environment variables
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/

# Set TLS certificates
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

# Enable TLS
export CORE_PEER_TLS_ENABLED=true

# Ensure throughput_res.txt exists
touch throughput_res.txt

# Define test parameters
NUM_CONCURRENT_DEFAULT=5
MAX_RETRIES=3
TIMEOUT_SECONDS=30

# Color variables for output formatting
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Storage for record IDs and owners (for authorization tracking)
record_ids=()
record_owners=()

# Trap to handle clean up of background processes and temporary files
trap 'kill $(jobs -p) 2>/dev/null; exit' INT TERM EXIT

# Function to set up environment for a hospital
setup_hospital_env() {
  local hospital=$1
  export CORE_PEER_LOCALMSPID="Org1MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
  export CORE_PEER_ADDRESS=localhost:7051
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/${hospital}@org1.example.com/msp
  echo "Environment set for $hospital"
}

# Function to set up environment for a doctor
setup_doctor_env() {
  local doctor=$1
  export CORE_PEER_LOCALMSPID="Org1MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
  export CORE_PEER_ADDRESS=localhost:7051
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/${doctor}@org1.example.com/msp
  echo "Environment set for $doctor"
}

# Function to set up environment for a patient
setup_patient_env() {
  local patient=$1
  export CORE_PEER_LOCALMSPID="Org2MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
  export CORE_PEER_ADDRESS=localhost:9051
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/${patient}@org2.example.com/msp
  echo "Environment set for $patient"
}

# Function to perform a ReadRecord operation
perform_read_record() {
  local user=$1
  local user_type=$2
  local record_id=$3
  local result_file=$4
  
  # Set up environment according to user type
  if [ "$user_type" = "hospital" ]; then
    setup_hospital_env "$user"
  elif [ "$user_type" = "doctor" ]; then
    setup_doctor_env "$user"
  else
    setup_patient_env "$user"
  fi
  
  local start_time=$(date +%s.%N)
  
  # Execute ReadRecord transaction
  if peer chaincode query -C emrchannel -n emr \
    -c "{\"Args\":[\"ReadRecord\",\"$record_id\"]}" \
    2>&1 > /dev/null; then
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Output result to file
    echo "SUCCESS $record_id $duration" >> $result_file
    return 0
  else
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Output failure to file
    echo "FAILURE $record_id $duration" >> $result_file
    return 1
  fi
}

# Function to set record owner
set_record_owner() {
  local record_id=$1
  local owner=$2
  
  # Check if record already exists
  for i in "${!record_ids[@]}"; do
    if [ "${record_ids[$i]}" = "$record_id" ]; then
      # Update existing record
      record_owners[$i]="$owner"
      return
    fi
  done
  
  # Add new record
  record_ids+=("$record_id")
  record_owners+=("$owner")
}

# Function to get record owner
get_record_owner() {
  local record_id=$1
  
  for i in "${!record_ids[@]}"; do
    if [ "${record_ids[$i]}" = "$record_id" ]; then
      echo "${record_owners[$i]}"
      return 0
    fi
  done
  
  # Record not found
  echo ""
  return 1
}
# Function to check if a user is registered in the chaincode
check_user_registered() {
  local user_type=$1
  local user_num=$2
  local org_num
  
  if [ "$user_type" = "patient" ]; then
    org_num=2
  else
    org_num=1
  fi
  
  # Set up environment for query
  export CORE_PEER_LOCALMSPID="Org${org_num}MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org${org_num}.example.com/tlsca/tlsca.org${org_num}.example.com-cert.pem
  export CORE_PEER_ADDRESS=localhost:$((7051 + (org_num-1)*2000))
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org${org_num}.example.com/users/${user_type}${user_num}@org${org_num}.example.com/msp

  # Check if user exists in chaincode
  if peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\", \"${user_type}${user_num}@org${org_num}.example.com\"]}" &>/dev/null; then
    return 0
  else
    return 1
  fi
}

# Function to register a new user
register_user() {
  local user_type=$1
  local user_num=$2
  local add_script
  local org_num
  local error_file="register_error.tmp"
  
  # Ensure cleanup happens even on unexpected exit
  trap 'rm -f "$error_file" 2>/dev/null || true' EXIT
  
  if [ "$user_type" = "patient" ]; then
    org_num=2
    add_script="./p_add.sh"
  elif [ "$user_type" = "doctor" ]; then
    org_num=1
    add_script="./d_add.sh"
  elif [ "$user_type" = "hospital" ]; then
    org_num=1
    add_script="./h_add.sh"
  else
    echo "Invalid user type: $user_type"
    return 1
  fi

  # Check if the add script exists and is executable
  if [ -f "$add_script" ] && [ -x "$add_script" ]; then
    echo "Running $add_script to register ${user_type}${user_num}"
    
    # Execute the add script with the user number, capturing stderr separately
    if ! $add_script "$user_num" >/dev/null 2>"$error_file"; then
      local error_msg
      error_msg=$(cat "$error_file" 2>/dev/null)
      echo "Failed to register ${user_type}${user_num}${error_msg:+: $error_msg}"
      # Don't return yet, let the verification step determine if it succeeded
    fi
  else
    echo "Registration script $add_script not found or not executable"
    return 1
  fi

  # Verify registration
  if check_user_registered "$user_type" "$user_num"; then
    echo "${user_type}${user_num} successfully registered"
    return 0
  else
    echo "Failed to register ${user_type}${user_num}"
    return 1
  fi
}

# Function to verify and setup users for testing
verify_setup_users() {
  local group_size=$1
  local users_per_type=$((group_size / 3))
  
  echo "Verifying user setup for group size $group_size ($users_per_type of each type)..."
  
  # Count currently registered users
  local hospital_count=0
  local doctor_count=0
  local patient_count=0
  
  for i in $(seq 1 8); do
    if check_user_registered "hospital" $i; then
      hospital_count=$((hospital_count + 1))
    fi
    if check_user_registered "doctor" $i; then
      doctor_count=$((doctor_count + 1))
    fi
    if check_user_registered "patient" $i; then
      patient_count=$((patient_count + 1))
    fi
  done
  
  echo "Found users: $hospital_count hospitals, $doctor_count doctors, $patient_count patients"
  
  # Check if we have enough users already registered
  if [ $hospital_count -ge $users_per_type ] && [ $doctor_count -ge $users_per_type ] && [ $patient_count -ge $users_per_type ]; then
    echo "User setup verified successfully for group size $group_size"
    return 0
  fi
  
  # Register additional users if needed
  local reg_success=true
  for i in $(seq 1 $users_per_type); do
    if ! check_user_registered "hospital" $i; then
      echo "Registering hospital$i..."
      if ! register_user "hospital" $i; then
        reg_success=false
        sleep 2  # Add delay after failed registration
      fi
    fi
    
    if ! check_user_registered "doctor" $i; then
      echo "Registering doctor$i..."
      if ! register_user "doctor" $i; then
        reg_success=false
        sleep 2  # Add delay after failed registration
      fi
    fi
    
    if ! check_user_registered "patient" $i; then
      echo "Registering patient$i..."
      if ! register_user "patient" $i; then
        reg_success=false
        sleep 2  # Add delay after failed registration
      fi
    fi
    
    # Add small delay between registration attempts
    sleep 1
  done
  
  if [ "$reg_success" = false ]; then
    echo "Warning: Some users could not be registered. Verifying final setup..."
  fi
  
  # Verify final user count
  hospital_count=0
  doctor_count=0
  patient_count=0
  
  for i in $(seq 1 $users_per_type); do
    if check_user_registered "hospital" $i; then
      hospital_count=$((hospital_count + 1))
    fi
    if check_user_registered "doctor" $i; then
      doctor_count=$((doctor_count + 1))
    fi
    if check_user_registered "patient" $i; then
      patient_count=$((patient_count + 1))
    fi
  done
  
  echo "Final user count: $hospital_count hospitals, $doctor_count doctors, $patient_count patients"
  
  if [ $hospital_count -ge $users_per_type ] && [ $doctor_count -ge $users_per_type ] && [ $patient_count -ge $users_per_type ]; then
    echo "User setup verified successfully for group size $group_size"
    return 0
  else
    echo "Error: Failed to set up required users for group size $group_size"
    echo "Required per type: $users_per_type"
    echo "Available: Hospitals: $hospital_count, Doctors: $doctor_count, Patients: $patient_count"
    return 1
  fi
}
# Function to perform a CreateRecord operation
perform_create_record() {
  local doctor=$1
  local patient=$2
  local hospital=$3
  local record_id=$4
  local result_file=$5
  
  setup_doctor_env "$doctor"
  
  local start_time=$(date +%s.%N)
  
  # Execute CreateRecord transaction
  if peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c "{\"Args\":[\"CreateRecord\",\"$record_id\",\"$patient@org2.example.com\",\"$doctor@org1.example.com\",\"$hospital@org1.example.com\",\"Throughput test record\"]}" \
    --waitForEvent 2>&1 > /dev/null; then
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Output result to file
    echo "SUCCESS $record_id $duration" >> $result_file
    return 0
  else
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Output failure to file
    echo "FAILURE $record_id $duration" >> $result_file
    return 1
  fi
}

# Function to run operations in parallel
run_parallel_operations() {
  local operation=$1
  local total_ops=$2
  local concurrency=$3
  local group_size=$4
  local timestamp=$5
  local temp_dir=$(mktemp -d)
  local result_file="${temp_dir}/results.txt"
  local pids=()
  local hospitals_per_type=$((group_size / 3))
  local doctors_per_type=$((group_size / 3))
  local patients_per_type=$((group_size / 3))
  
  echo "Running $total_ops $operation operations with concurrency $concurrency for group size $group_size"
  
  # Create temporary result file
  touch "$result_file"
  
  local start_time=$(date +%s.%N)
  
  # Distribute operations across concurrent processes
  for ((i=1; i<=$total_ops; i++)); do
    if [ "$operation" = "CreateRecord" ]; then
      # Randomly select users for the operation
      local doctor_idx=$(( (RANDOM % doctors_per_type) + 1 ))
      local patient_idx=$(( (RANDOM % patients_per_type) + 1 ))
      local hospital_idx=$(( (RANDOM % hospitals_per_type) + 1 ))
      local record_id="EMR_TH_${timestamp}_${i}"
      
      # Run operation in background
      perform_create_record "doctor${doctor_idx}" "patient${patient_idx}" "hospital${hospital_idx}" "$record_id" "$result_file" &
      pids+=($!)
      
    elif [ "$operation" = "ReadRecord" ]; then
      # Get an existing record ID from our collection
      if [ ${#record_ids[@]} -eq 0 ]; then
        echo "No records available for ReadRecord test"
        return 1
      fi
      
      local idx=$(( RANDOM % ${#record_ids[@]} ))
      local record_id="${record_ids[$idx]}"
      local doctor=${record_owners[$idx]}
      
      # Run operation in background
      perform_read_record "$doctor" "doctor" "$record_id" "$result_file" &
      pids+=($!)
      
    elif [ "$operation" = "ShareRecord" ]; then
      # Get an existing record ID from our collection
      if [ ${#record_ids[@]} -eq 0 ]; then
        echo "No records available for ShareRecord test"
        return 1
      fi
      
      local idx=$(( RANDOM % ${#record_ids[@]} ))
      local record_id="${record_ids[$idx]}"
      local owner_doctor=${record_owners[$idx]}
      
      # Select a target doctor different from the owner
      local target_doctor_idx=$(( (RANDOM % doctors_per_type) + 1 ))
      while [ "doctor${target_doctor_idx}" = "$owner_doctor" ] && [ $doctors_per_type -gt 1 ]; do
        target_doctor_idx=$(( (RANDOM % doctors_per_type) + 1 ))
      done
      
      # Run operation in background
      perform_share_record "$owner_doctor" "doctor${target_doctor_idx}" "doctor" "$record_id" "$result_file" &
      pids+=($!)
    fi
    
    # Control concurrency by waiting for some processes to finish
    if [ ${#pids[@]} -ge $concurrency ]; then
      wait ${pids[0]}
      pids=("${pids[@]:1}")
    fi
    
    # Small delay to prevent overwhelming the system
    sleep 0.1
  done
  
  # Wait for all remaining processes to finish
  echo "Waiting for all operations to complete..."
  for pid in "${pids[@]}"; do
    wait $pid 2>/dev/null || true
  done
  # Clear pids array
  pids=()
  
  local end_time=$(date +%s.%N)
  local total_duration=$(echo "$end_time - $start_time" | bc)
  
  # Process results
  local successful=0
  local failed=0
  local total_op_time=0
  
  while IFS= read -r line; do
    if [[ $line == SUCCESS* ]]; then
      successful=$((successful + 1))
      # Extract the duration from the line (third field)
      local duration=$(echo "$line" | awk '{print $3}')
      total_op_time=$(echo "$total_op_time + $duration" | bc)
    elif [[ $line == FAILURE* ]]; then
      failed=$((failed + 1))
    fi
  done < "$result_file"
  
  local total_ops=$((successful + failed))
  local success_rate=0
  if [ $total_ops -gt 0 ]; then
    success_rate=$(echo "scale=2; ($successful * 100) / $total_ops" | bc)
  fi
  
  local avg_op_time=0
  if [ $successful -gt 0 ]; then
    avg_op_time=$(echo "scale=6; $total_op_time / $successful" | bc)
  fi
  
  local throughput=0
  if [ $(echo "$total_duration > 0" | bc) -eq 1 ]; then
    throughput=$(echo "scale=2; $successful / $total_duration" | bc)
  fi
  
  # Output results to console
  echo -e "${GREEN}=== $operation Results ===${NC}"
  echo "Total operations: $total_ops ($successful successful, $failed failed)"
  echo "Success rate: ${success_rate}%"
  echo "Average operation time: ${avg_op_time} seconds"
  echo "Total execution time: ${total_duration} seconds"
  echo "Throughput: ${throughput} transactions per second"
  
  # Store results in throughput_res.txt
  echo "----------------------------------------" >> throughput_res.txt
  echo "Timestamp: $(date "+%Y-%m-%d %H:%M:%S")" >> throughput_res.txt
  echo "Operation: $operation" >> throughput_res.txt
  echo "Group size: $group_size" >> throughput_res.txt
  echo "Concurrency: $concurrency" >> throughput_res.txt
  echo "Total operations: $total_ops ($successful successful, $failed failed)" >> throughput_res.txt
  echo "Success rate: ${success_rate}%" >> throughput_res.txt
  echo "Average operation time: ${avg_op_time} seconds" >> throughput_res.txt
  echo "Total execution time: ${total_duration} seconds" >> throughput_res.txt
  echo "Throughput: ${throughput} transactions per second" >> throughput_res.txt
  
  # Clean up temporary directory
  rm -rf "$temp_dir"
  
  # Return throughput value
  echo "$throughput"
}

# Function to prepare test environment and run throughput tests for a specific group size
run_throughput_test() {
  local group_size=$1
  local concurrency=$2
  local timestamp=$(date +%s)
  
  echo -e "\n${YELLOW}===== Preparing for throughput test with $group_size users =====${NC}"
  
  # Verify user setup
  if ! verify_setup_users $group_size; then
    echo -e "${RED}Failed to verify user setup for group size $group_size. Skipping tests.${NC}"
    return 1
  fi
  
  echo -e "\n${YELLOW}===== Running CreateRecord throughput test =====${NC}"
  local create_ops=20
  local create_tps=$(run_parallel_operations "CreateRecord" $create_ops $concurrency $group_size $timestamp)
  
  # Small delay between test types
  sleep 5
  
  echo -e "\n${YELLOW}===== Running ReadRecord throughput test =====${NC}"
  local read_ops=50
  local read_tps=$(run_parallel_operations "ReadRecord" $read_ops $concurrency $group_size $timestamp)
  
  # Small delay between test types
  sleep 5
  
  echo -e "\n${YELLOW}===== Running ShareRecord throughput test =====${NC}"
  local share_ops=20
  local share_tps=$(run_parallel_operations "ShareRecord" $share_ops $concurrency $group_size $timestamp)
  
  # Output summary of results
  echo -e "\n${GREEN}===== Throughput Test Summary for $group_size Users =====${NC}"
  echo "CreateRecord TPS: $create_tps"
  echo "ReadRecord TPS: $read_tps"
  echo "ShareRecord TPS: $share_tps"
  echo "Test timestamp: $timestamp"
  
  # Store summary in throughput_res.txt
  echo "========================================" >> throughput_res.txt
  echo "SUMMARY - Group Size: $group_size, Concurrency: $concurrency" >> throughput_res.txt
  echo "Timestamp: $(date "+%Y-%m-%d %H:%M:%S")" >> throughput_res.txt
  echo "CreateRecord TPS: $create_tps" >> throughput_res.txt
  echo "ReadRecord TPS: $read_tps" >> throughput_res.txt
  echo "ShareRecord TPS: $share_tps" >> throughput_res.txt
  echo "========================================" >> throughput_res.txt
}

# Function to calculate concurrent users based on group size
calculate_concurrency() {
  local group_size=$1
  local concurrency
  
  # Adjust concurrency based on group size
  if [ $group_size -le 6 ]; then
    concurrency=3
  elif [ $group_size -le 12 ]; then
    concurrency=5
  elif [ $group_size -le 18 ]; then
    concurrency=8
  else
    concurrency=10
  fi
  
  echo $concurrency
}

# Function to perform a ShareRecord operation
perform_share_record() {
  local doctor=$1
  local target_user=$2
  local target_type=$3
  local record_id=$4
  local result_file=$5
  
  setup_doctor_env "$doctor"
  
  local target_domain
  if [ "$target_type" = "patient" ]; then
    target_domain="org2.example.com"
  else
    target_domain="org1.example.com"
  fi
  
  local start_time=$(date +%s.%N)
  
  # Execute ShareRecord transaction
  if peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c "{\"Args\":[\"ShareRecord\",\"$record_id\",\"$target_user@$target_domain\",\"$target_type\"]}" \
    --waitForEvent 2>&1 > /dev/null; then
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Output result to file
    echo "SUCCESS $record_id $duration" >> $result_file
    return 0
  else
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Output failure to file
    echo "FAILURE $record_id $duration" >> $result_file
    return 1
  fi
}

# Main execution starts here
echo "=== EMR Network Throughput Test ==="
echo "Starting tests at $(date)"

# Check requirements before starting
check_requirements

# Create header in results file if it doesn't exist or is empty
if [ ! -s throughput_res.txt ]; then
  echo "=== EMR Network Throughput Test Results ===" > throughput_res.txt
  echo "Started: $(date)" >> throughput_res.txt
  echo "----------------------------------------" >> throughput_res.txt
fi

# Run tests for different user group sizes
for size in 6 12 18 24; do
  concurrency=$(calculate_concurrency $size)
  echo -e "\n${YELLOW}===== Testing with $size users and concurrency $concurrency =====${NC}"
  run_throughput_test $size $concurrency
  
  # Add a delay between group sizes to prevent network congestion
  echo "Waiting before next test group..."
  sleep 10
done

echo -e "\n${GREEN}All throughput tests completed!${NC}"
echo "Results stored in throughput_res.txt"
