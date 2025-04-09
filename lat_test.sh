#!/bin/bash

# Exit on first error
set -e

# Import utils
. scripts/utils.sh

# Set environment variables
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/

# Set TLS certificates
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

# Enable TLS
export CORE_PEER_TLS_ENABLED=true

# Ensure latency_res.txt exists
touch latency_res.txt

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
  local script_name
  local org_num
  
  if [ "$user_type" = "patient" ]; then
    org_num=2
    script_name="p_reg.sh"
  elif [ "$user_type" = "doctor" ]; then
    org_num=1
    script_name="d_reg.sh"
  elif [ "$user_type" = "hospital" ]; then
    org_num=1
    script_name="h_reg.sh"
  else
    echo "Invalid user type: $user_type"
    return 1
  fi

  # Check if user MSP directory exists
  if [ -d "${PWD}/organizations/peerOrganizations/org${org_num}.example.com/users/${user_type}${user_num}@org${org_num}.example.com" ]; then
    echo "User credentials for ${user_type}${user_num} exist, attempting direct registration"
    
    # Set up environment for registration
    export CORE_PEER_LOCALMSPID="Org${org_num}MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org${org_num}.example.com/tlsca/tlsca.org${org_num}.example.com-cert.pem
    export CORE_PEER_ADDRESS=localhost:$((7051 + (org_num-1)*2000))
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org${org_num}.example.com/users/${user_type}${user_num}@org${org_num}.example.com/msp
    
    # Register user directly with chaincode
    peer chaincode invoke -o localhost:7050 \
      --ordererTLSHostnameOverride orderer.example.com \
      --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
      -C emrchannel -n emr \
      --peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
      --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem \
      -c '{"Args":["RegisterUser"]}' --waitForEvent &>/dev/null || true
  else
    # Use registration script if available
    if [ -f "$script_name" ] && [ -x "$script_name" ]; then
      echo "Running $script_name to register ${user_type}${user_num}"
      # Capture output but ignore errors about user already registered
      ./$script_name &>/dev/null || true
    else
      echo "Registration script $script_name not found or not executable"
      return 1
    fi
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
  
  # Check if we have too many users already registered
  if [ $hospital_count -gt $users_per_type ] || [ $doctor_count -gt $users_per_type ] || [ $patient_count -gt $users_per_type ]; then
    echo "Error: Too many users already registered for test size $group_size"
    echo "Maximum allowed per type: $users_per_type"
    echo "Found: Hospitals: $hospital_count, Doctors: $doctor_count, Patients: $patient_count"
    return 1
  fi
  
  # Register additional users if needed
  local reg_success=true
  for i in $(seq 1 $users_per_type); do
    if ! check_user_registered "hospital" $i; then
      echo "Registering hospital$i..."
      if ! register_user "hospital" $i; then
        reg_success=false
      fi
    fi
    
    if ! check_user_registered "doctor" $i; then
      echo "Registering doctor$i..."
      if ! register_user "doctor" $i; then
        reg_success=false
      fi
    fi
    
    if ! check_user_registered "patient" $i; then
      echo "Registering patient$i..."
      if ! register_user "patient" $i; then
        reg_success=false
      fi
    fi
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

# Function to setup environment for a hospital
setup_hospital_env() {
  local hospital=$1
  export CORE_PEER_LOCALMSPID="Org1MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
  export CORE_PEER_ADDRESS=localhost:7051
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/${hospital}@org1.example.com/msp
  echo "Environment set for $hospital"
}

# Function to setup environment for a doctor
setup_doctor_env() {
  local doctor=$1
  export CORE_PEER_LOCALMSPID="Org1MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
  export CORE_PEER_ADDRESS=localhost:7051
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/${doctor}@org1.example.com/msp
  echo "Environment set for $doctor"
}

# Function to setup environment for a patient
setup_patient_env() {
  local patient=$1
  export CORE_PEER_LOCALMSPID="Org2MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
  export CORE_PEER_ADDRESS=localhost:9051
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/${patient}@org2.example.com/msp
  echo "Environment set for $patient"
}

# Function to store record mappings for authorization tracking
record_ids=()
record_owners=()

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
# Function to measure latency for CreateRecord operation
measure_create_record() {
  local doctor=$1
  local patient=$2
  local hospital=$3
  local record_id=$4
  local attempt=$5
  
  setup_doctor_env "$doctor"
  
  echo "Testing CreateRecord: Doctor $doctor creating record $record_id for patient $patient at hospital $hospital (Attempt $attempt)"
  
  # Measure start time
  local start_time=$(date +%s.%N)
  
  # Execute CreateRecord transaction
  if ! peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c "{\"Args\":[\"CreateRecord\",\"$record_id\",\"$patient@org2.example.com\",\"$doctor@org1.example.com\",\"$hospital@org1.example.com\",\"Latency test record\"]}" \
    --waitForEvent 2>&1 > /dev/null; then
    echo "Failed to create record $record_id"
    return 1
  fi
  
  # Store record ownership for later authorization checks
  set_record_owner "$record_id" "$doctor"
  
  # Measure end time
  local end_time=$(date +%s.%N)
  
  # Calculate latency
  local latency=$(echo "$end_time - $start_time" | bc)
  echo "$latency"
}

# Function to measure latency for ReadRecord operation
measure_read_record() {
  local user=$1
  local user_type=$2
  local record_id=$3
  local attempt=$4
  
  # Skip if the user is not authorized for this record
  local record_owner=$(get_record_owner "$record_id")
  if [ "$user_type" = "doctor" ] && [ -n "$record_owner" ] && [ "$record_owner" != "$user" ]; then
    echo "Skipping unauthorized read: $user is not the owner of record $record_id"
    return 2
  fi
  
  if [ "$user_type" = "hospital" ] || [ "$user_type" = "doctor" ]; then
    setup_doctor_env "$user"
  else
    setup_patient_env "$user"
  fi
  
  echo "Testing ReadRecord: $user_type $user reading record $record_id (Attempt $attempt)"
  
  # Measure start time
  local start_time=$(date +%s.%N)
  
  # Execute ReadRecord transaction
  if ! peer chaincode query -C emrchannel -n emr \
    -c "{\"Args\":[\"ReadRecord\",\"$record_id\"]}" \
    2>&1 > /dev/null; then
    echo "Failed to read record $record_id by $user_type $user"
    return 1
  fi
  
  # Measure end time
  local end_time=$(date +%s.%N)
  
  # Calculate latency
  local latency=$(echo "$end_time - $start_time" | bc)
  echo "$latency"
}

# Function to measure latency for ShareRecord operation
measure_share_record() {
  local doctor=$1
  local target_user=$2
  local target_type=$3
  local record_id=$4
  local attempt=$5
  
  # Skip if the doctor is not authorized for this record
  local record_owner=$(get_record_owner "$record_id")
  if [ -n "$record_owner" ] && [ "$record_owner" != "$doctor" ]; then
    echo "Skipping unauthorized share: $doctor is not the owner of record $record_id"
    return 2
  fi
  
  setup_doctor_env "$doctor"
  
  local target_domain
  if [ "$target_type" = "patient" ]; then
    target_domain="org2.example.com"
  else
    target_domain="org1.example.com"
  fi
  
  echo "Testing ShareRecord: Doctor $doctor sharing record $record_id with $target_type $target_user (Attempt $attempt)"
  
  # Measure start time
  local start_time=$(date +%s.%N)
  
  # Execute ShareRecord transaction
  if ! peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c "{\"Args\":[\"ShareRecord\",\"$record_id\",\"$target_user@$target_domain\",\"$target_type\"]}" \
    --waitForEvent 2>&1 > /dev/null; then
    echo "Failed to share record $record_id with $target_type $target_user"
    return 1
  fi
  
  # Measure end time
  local end_time=$(date +%s.%N)
  
  # Calculate latency
  local latency=$(echo "$end_time - $start_time" | bc)
  echo "$latency"
}

# Function to run latency tests for a specific group size
run_latency_test() {
  local total_users=$1
  local hospitals=$((total_users / 3))
  local doctors=$((total_users / 3))
  local patients=$((total_users / 3))
  local timestamp=$(date "+%Y-%m-%d %H:%M:%S")
  
  echo "Running tests for $total_users users ($hospitals hospitals, $doctors doctors, $patients patients) at $timestamp"
  
  # Initialize arrays for latency measurements
  local create_latencies=()
  local read_latencies=()
  local share_latencies=()
  
  # Number of repetitions for each test
  local repetitions=5
  
  # Test CreateRecord operation
  for i in $(seq 1 $repetitions); do
    # Randomly select users for each test
    local hospital_idx=$((RANDOM % hospitals + 1))
    local doctor_idx=$((RANDOM % doctors + 1))
    local patient_idx=$((RANDOM % patients + 1))
    
    local record_id="EMR_LAT_${total_users}_${i}"
    local hospital="hospital${hospital_idx}"
    local doctor="doctor${doctor_idx}"
    local patient="patient${patient_idx}"
    
    local latency=$(measure_create_record "$doctor" "$patient" "$hospital" "$record_id" "$i")
    if [[ "$latency" == *"Failed"* ]]; then
      echo "CreateRecord operation failed: $latency"
    else
      create_latencies+=("$latency")
      echo "CreateRecord latency: $latency seconds"
    fi
    sleep 2
  done
  
  # Test ReadRecord operation
  for i in $(seq 1 $repetitions); do
    # Use the doctor who created the record
    local record_id="EMR_LAT_${total_users}_$i"
    local record_owner=$(get_record_owner "$record_id")
    local doctor=${record_owner:-"doctor1"}
    
    local latency=$(measure_read_record "$doctor" "doctor" "$record_id" "$i")
    if [[ "$latency" == *"Failed"* ]] || [[ "$latency" == *"Skipping"* ]]; then
      echo "ReadRecord operation skipped or failed: $latency"
    else
      read_latencies+=("$latency")
      echo "ReadRecord latency: $latency seconds"
    fi
    sleep 2
  done
  
  # Test ShareRecord operation
  for i in $(seq 1 $repetitions); do
    # Use the doctor who created the record, and share with another doctor
    local record_id="EMR_LAT_${total_users}_$i"
    local record_owner=$(get_record_owner "$record_id")
    local doctor=${record_owner:-"doctor1"}
    
    # Select a different doctor to share with
    local target_doctor_idx=$(( (doctor_idx % doctors) + 1 ))
    [ "$target_doctor_idx" -eq "${doctor:6}" ] && target_doctor_idx=$(( (target_doctor_idx % doctors) + 1 ))
    local target_doctor="doctor${target_doctor_idx}"
    
    local latency=$(measure_share_record "$doctor" "$target_doctor" "doctor" "$record_id" "$i")
    if [[ "$latency" == *"Failed"* ]] || [[ "$latency" == *"Skipping"* ]]; then
      echo "ShareRecord operation skipped or failed: $latency"
    else
      share_latencies+=("$latency")
      echo "ShareRecord latency: $latency seconds"
    fi
    sleep 2
  done
  
  # Calculate average latencies
  local create_avg=0
  local read_avg=0
  local share_avg=0
  
  # Calculate average for CreateRecord
  if [ ${#create_latencies[@]} -gt 0 ]; then
    local sum=0
    for latency in "${create_latencies[@]}"; do
      sum=$(echo "$sum + $latency" | bc)
    done
    create_avg=$(echo "scale=6; $sum / ${#create_latencies[@]}" | bc)
  fi
  
  # Calculate average for ReadRecord
  if [ ${#read_latencies[@]} -gt 0 ]; then
    local sum=0
    for latency in "${read_latencies[@]}"; do
      sum=$(echo "$sum + $latency" | bc)
    done
    read_avg=$(echo "scale=6; $sum / ${#read_latencies[@]}" | bc)
  fi
  
  # Calculate average for ShareRecord
  if [ ${#share_latencies[@]} -gt 0 ]; then
    local sum=0
    for latency in "${share_latencies[@]}"; do
      sum=$(echo "$sum + $latency" | bc)
    done
    share_avg=$(echo "scale=6; $sum / ${#share_latencies[@]}" | bc)
  fi
  
  # Append results to latency_res.txt
  echo "----------------------------------------" >> latency_res.txt
  echo "Timestamp: $timestamp" >> latency_res.txt
  echo "Users: $total_users ($hospitals hospitals, $doctors doctors, $patients patients)" >> latency_res.txt
  echo "CreateRecord average latency: $create_avg seconds (${#create_latencies[@]} successful operations)" >> latency_res.txt
  echo "ReadRecord average latency: $read_avg seconds (${#read_latencies[@]} successful operations)" >> latency_res.txt
  echo "ShareRecord average latency: $share_avg seconds (${#share_latencies[@]} successful operations)" >> latency_res.txt
  
  # Print summary
  echo "Test results for $total_users users:"
  echo "  CreateRecord average latency: $create_avg seconds"
  echo "  ReadRecord average latency: $read_avg seconds"
  echo "  ShareRecord average latency: $share_avg seconds"
  echo "Results appended to latency_res.txt"
  echo ""
}

# Main execution starts here
echo "=== EMR Network Latency Test ==="
echo "Starting tests at $(date)"

# Create header in results file if it doesn't exist or is empty
if [ ! -s latency_res.txt ]; then
  echo "=== EMR Network Latency Test Results ===" > latency_res.txt
  echo "Started: $(date)" >> latency_res.txt
  echo "----------------------------------------" >> latency_res.txt
fi

# Run tests for different user group sizes
for size in 6 12 18 24; do
  echo "===== Preparing for test with $size users ====="
  if verify_setup_users $size; then
    echo "===== Running tests with $size users ====="
    run_latency_test $size
  else
    echo "Skipping tests for $size users due to setup failure"
    # Optionally exit here if you want to stop at first failure
    # exit 1
  fi
done

echo "All latency tests completed successfully!"
echo "Results stored in latency_res.txt"

# Make the script executable
chmod +x lat_test.sh

