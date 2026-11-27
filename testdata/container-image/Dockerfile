FROM scratch AS image
ARG DISTRO
LABEL distro=$DISTRO
COPY distros/$DISTRO/distro.env /distro.env

# GREETING reaches the build only when distro.env lists it in BUILD_ARGS
FROM busybox:1.37.0 AS test
ARG DISTRO
ARG GREETING
COPY --from=image /distro.env /distro.env
RUN [ -n "$DISTRO" ] && [ "$GREETING" = "$(sed -n 's/^GREETING=//p' /distro.env)" ]
