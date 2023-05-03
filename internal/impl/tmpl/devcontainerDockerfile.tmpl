FROM debian:stable-slim

# Step 1: Installing dependencies
RUN apt-get update
RUN apt-get -y install bash binutils git xz-utils wget sudo

# Step 2: Setting up devbox user
ENV DEVBOX_USER=devbox
RUN adduser $DEVBOX_USER
RUN usermod -aG sudo $DEVBOX_USER
RUN echo "devbox ALL=(ALL:ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/$DEVBOX_USER

USER $DEVBOX_USER

# Step 3: Installing devbox
RUN wget --quiet --output-document=/dev/stdout https://get.jetpack.io/devbox   | bash -s -- -f
RUN chown -R "${DEVBOX_USER}:${DEVBOX_USER}" /usr/local/bin/devbox

# Step 4: Installing Nix
RUN wget --output-document=/dev/stdout https://nixos.org/nix/install | sh -s -- --no-daemon
RUN . ~/.nix-profile/etc/profile.d/nix.sh
# updating PATH
ENV PATH="/home/${DEVBOX_USER}/.nix-profile/bin:/home/${DEVBOX_USER}/.devbox/nix/profile/default/bin:${PATH}"

# Step 5: Installing your devbox project
WORKDIR /code
RUN sudo chown $DEVBOX_USER:root /code
COPY devbox.json devbox.json
RUN devbox install
CMD ["devbox", "shell"]
