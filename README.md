# dragonboard 410c

## Create the recovery image for dragonboard

#### Build requirements
1. Install GO language, setup GOPATH
2. Install go-flags
 * go get github.com/jessevdk/go-flags
3. Install recovery process libraries
 * go get github.com/Lyoncore/ubuntu-recovery-rplib
 * go get github.com/Lyoncore/ubuntu-recovery-rpoem

#### Build Steps
1. Build factory_data and image
 * make
