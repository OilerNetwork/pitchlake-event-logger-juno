# Stage 1: Build golang dependencies and binaries
FROM ubuntu:24.10 AS build

ARG VM_DEBUG

RUN apt-get -qq update && \
    apt-get -qq install curl build-essential gcc git golang upx-ucl libjemalloc-dev libbz2-dev libjemalloc2 -y
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -q -y

WORKDIR /plugin

COPY . . 
RUN git submodule update --init --remote --recursive
RUN bash -c 'cd juno && source ~/.cargo/env && VM_DEBUG=${VM_DEBUG} make juno'

RUN pwd
RUN ls

# Then build the plugin
RUN go mod tidy
RUN make build

# Stage 2: Run Juno with the plugin
FROM ubuntu:24.10

# Install necessary runtime dependencies
RUN apt-get -qq update && \
    apt-get -qq install -y ca-certificates curl gawk grep libjemalloc-dev libjemalloc2 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV L1_URL=${L1_URL}
# Copy the Juno binary and the plugin from the build stage
COPY --from=build /plugin/db/migrations ./db/migrations
COPY --from=build /plugin/juno/build/juno ./build/
COPY --from=build /plugin/myplugin.so ./


# Run Juno with the plugin
CMD ["bash", "-c", "./build/juno --plugin-path myplugin.so --http --http-port=6060 --http-host=0.0.0.0 --network sepolia --rpc-cors-enable --eth-node ${L1_URL} --db-path=/snapshots/"]

