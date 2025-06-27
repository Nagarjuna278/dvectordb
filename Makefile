# Makefile for generating Go Protobuf and gRPC code

# Define variables for the proto file location and output settings
# This makes the command easier to read and modify later
PROTO_SRC_DIR := proto
PROTO_FILE_NAME := message.proto
FULL_PROTO_PATH := $(PROTO_SRC_DIR)/$(PROTO_FILE_NAME)
GO_OUTPUT_DIR := $(PROTO_SRC_DIR) # CORRECTED: Explicitly output to the 'proto' directory
GO_OPTS := paths=source_relative

# .PHONY declares targets that do not produce actual files of the same name.
# This ensures 'make' always runs these commands when invoked,
# even if a file named 'gen-proto' or 'all' or 'clean' exists.
.PHONY: all gen-proto clean

# Target to generate Go Protobuf and gRPC code
# This target will be executed when you run 'make gen-proto'
gen-proto:
	@echo "Generating Go Protobuf and gRPC code for $(FULL_PROTO_PATH)..."
	# The command below MUST be indented with a single TAB character
	protoc --proto_path=$(PROTO_SRC_DIR) \
	       --go_out=$(GO_OUTPUT_DIR) --go_opt=$(GO_OPTS) \
	       --go-grpc_out=$(GO_OUTPUT_DIR) --go-grpc_opt=$(GO_OPTS) \
	       $(FULL_PROTO_PATH)
	@echo "Go proto generation complete."

# Target to clean up generated Go protobuf files
# This target will be executed when you run 'make clean'
clean:
	@echo "Cleaning up generated Go proto files..."
	# The command below MUST be indented with a single TAB character
	rm -f $(PROTO_SRC_DIR)/*.pb.go $(PROTO_SRC_DIR)/*_grpc.pb.go
	@echo "Clean up complete."

# You can also set a default target (e.g., 'all')
# If you just type 'make', it will run the 'gen-proto' target
all: gen-proto
