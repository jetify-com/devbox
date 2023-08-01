FROM debian:stable-slim

# Step 1: Installing dependencies
RUN apt-get update
RUN apt-get -y install bash binutils git xz-utils wget sudo

# Step 2: Installing Nix
RUN wget --output-document=/dev/stdout https://nixos.org/nix/install | sh -s -- --daemon
RUN . ~/.nix-profile/etc/profile.d/nix.sh
ENV PATH="/root/.nix-profile/bin:$PATH"

# Step 3: Installing devbox
RUN wget --quiet --output-document=/dev/stdout https://get.jetpack.io/devbox   | bash -s -- -f

# Step 4: Installing your devbox project
WORKDIR /code
COPY devbox.json devbox.json

# Step 6: Copying local flakes directories
COPY emulator emulator

# Step 7: copy code
COPY requirements.txt requirements.txt
COPY main.py main.py


RUN devbox run -- echo "Installed Packages."

CMD ["devbox", "shell"]
