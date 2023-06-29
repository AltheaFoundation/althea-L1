FROM fedora:37
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin
RUN dnf install -y git make gcc gcc-c++ which iproute iputils procps-ng vim-minimal tmux net-tools htop tar jq npm openssl-devel perl rust cargo golang wget
# the actual source code for this repo, this **only** includes checked in files!
# this is a bit of a pain but it does speed things up a lot
ADD althea.tar.gz /
# build the test runner specifically to cache a release artifact
RUN pushd /althea/integration_tests && PATH=$PATH:$HOME/.cargo/bin cargo build --bin test-runner --release
# generate artifacts for the ethereum contracts
RUN pushd /althea/solidity/ && HUSKY_SKIP_INSTALL=1 npm install && npm run typechain
# The althea bin now depends on the output of the contracts
RUN pushd /althea && make contracts
# build the althea chain binary
RUN pushd /althea/ && PATH=$PATH:/usr/local/go/bin GOPROXY=https://proxy.golang.org make && PATH=$PATH:/usr/local/go/bin make install
RUN git config --global --add safe.directory /althea
