FROM fedora:40
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin
RUN dnf install -y git make gcc gcc-c++ which iproute iputils procps-ng vim-minimal tmux net-tools htop tar jq npm openssl-devel perl rust cargo wget
COPY --from=golang:1.22 /usr/local/go/ /usr/local/go/
ENV PATH="/usr/local/go/bin:${PATH}"
# Download the althea gaia fork as a IBC test chain
ADD https://github.com/althea-net/ibc-test-chain/releases/download/v9.1.5/gaiad-v9.1.5-linux-amd64 /usr/bin/gaiad
# Setup Hermes for IBC connections between chains
ADD https://github.com/informalsystems/hermes/releases/download/v1.7.0/hermes-v1.7.0-x86_64-unknown-linux-gnu.tar.gz /tmp/
RUN cd /tmp/ && tar -xvf hermes-v1.7.0-x86_64-unknown-linux-gnu.tar.gz && mv hermes /usr/bin/
RUN mkdir /ibc-relayer-logs && touch /ibc-relayer-logs/hermes-logs && touch /ibc-relayer-logs/channel-creation

# the actual source code for this repo, this **only** includes checked in files!
# this is a bit of a pain but it does speed things up a lot
ADD althea.tar.gz /
RUN git clone https://github.com/AltheaFoundation/althea-dex.git /althea/solidity-dex
# build the test runner specifically to cache a release artifact
RUN pushd /althea/integration_tests && PATH=$PATH:$HOME/.cargo/bin cargo build --bin test-runner --release
# generate artifacts for the ethereum contracts
RUN pushd /althea/solidity/ && HUSKY_SKIP_INSTALL=1 npm install && npm run typechain
# generate artifacts for the dex contracts
RUN pushd /althea/solidity-dex/ && HUSKY_SKIP_INSTALL=1 npm install && npx hardhat compile
# The althea bin now depends on the output of the contracts
RUN pushd /althea && make
# build the althea chain binary
RUN pushd /althea/ && PATH=$PATH:/usr/local/go/bin GOPROXY=https://proxy.golang.org make && PATH=$PATH:/usr/local/go/bin make install
RUN git config --global --add safe.directory /althea
