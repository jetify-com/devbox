FROM alpine:3

# Setting up devbox user
ENV DEVBOX_USER=devbox
RUN adduser -h /home/$DEVBOX_USER -D -s /bin/bash $DEVBOX_USER
RUN addgroup sudo
RUN addgroup $DEVBOX_USER sudo
RUN echo " $DEVBOX_USER      ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers

# installing dependencies
RUN apk add --no-cache bash binutils git libstdc++ xz sudo

USER $DEVBOX_USER

# installing devbox
RUN wget --quiet --output-document=/dev/stdout https://get.jetpack.io/devbox | bash -s -- -f
RUN chown -R "${DEVBOX_USER}:${DEVBOX_USER}" /usr/local/bin/devbox

# nix installer script
RUN wget --quiet --output-document=/dev/stdout https://nixos.org/nix/install | sh -s -- --no-daemon
RUN . ~/.nix-profile/etc/profile.d/nix.sh
# updating PATH
ENV PATH="/home/${DEVBOX_USER}/.nix-profile/bin:/home/${DEVBOX_USER}/.devbox/nix/profile/default/bin:${PATH}"

WORKDIR /code
COPY devbox.json devbox.json
RUN devbox shell -- echo "Installing packages"
ENTRYPOINT ["devbox"]
CMD ['shell']

