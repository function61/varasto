# would use Alpine but libc not compatible with the smartmontools that we compiled
FROM ubuntu:latest

CMD ["/usr/bin/stosmart"]

ADD smartctl /usr/sbin/smartctl

ADD stosmart /usr/bin/stosmart

# Usage:
# $ docker run -d  \
# 	--cap-add SYS_RAWIO \
# 	--privileged \
# 	-v /dev:/dev \
# 	--label 'edgerouter.auth=public'
# 	--label 'traefik.frontend.rule=Host:smart.home.fn61.net' \
# 	joonas/stosmart
#
# (compose / Swarm doesn't work due to --privileged meaning something else and we'd have to
#  --device map each drive statically in a way that doesn't support hot plug)
# 
# /dev:/dev because otherwise /dev/disk/by-label/... not available
