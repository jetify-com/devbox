FROM debian:stable-slim

# Step 1: Installing dependencies
RUN apt-get update
RUN apt-get -y install bash binutils git xz-utils wget sudo vim patchelf

# Step 1.5: Setting up devbox user
ENV DEVBOX_USER=devbox
RUN adduser $DEVBOX_USER
RUN usermod -aG sudo $DEVBOX_USER
RUN echo "devbox ALL=(ALL:ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/$DEVBOX_USER
USER $DEVBOX_USER

# Step 2: Installing Nix
RUN wget --output-document=/dev/stdout https://nixos.org/nix/install | sh -s -- --no-daemon
RUN . ~/.nix-profile/etc/profile.d/nix.sh

ENV PATH="/home/${DEVBOX_USER}/.nix-profile/bin:$PATH"

# Step 3: Installing devbox
RUN wget --quiet --output-document=/dev/stdout https://get.jetpack.io/devbox   | bash -s -- -f
RUN chown -R "${DEVBOX_USER}:${DEVBOX_USER}" /usr/local/bin/devbox

# Step 4: Installing your devbox project
WORKDIR /code
COPY devbox.json devbox.json
COPY devbox.lock devbox.lock
COPY requirements.txt requirements.txt
COPY main.py main.py
COPY Dockerfile Dockerfile
RUN sudo chown -R "${DEVBOX_USER}:${DEVBOX_USER}" /code


RUN devbox run -- echo "Installed Packages."

CMD ["/bin/bash"]
