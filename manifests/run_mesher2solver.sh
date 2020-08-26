#! /bin/bash

set -xe

test -c /dev/fuse # mknod /dev/fuse c 10 229

cp /var/run/secrets/openshift.io/push/.dockercfg /tmp
(echo "{ \"auths\": " ; cat /var/run/secrets/openshift.io/push/.dockercfg ; echo "}") > /tmp/.dockercfg

AUTH="--tls-verify=false --authfile /tmp/.dockercfg"

IMG_STREAM=image-registry.openshift-image-registry.svc:5000/{{ .App.ObjectMeta.Namespace }}/specfem
cont=$(buildah $AUTH from $IMG_STREAM:mesher)

buildah run $cont bash -c 'echo "$(date) | Using BUILDAH --volume to build the solver  ..." >> /app/oc.build.log'

buildah run --volume /mnt/shared:/mnt/shared:rw,z $cont bash -c '\
        echo "$(date) | Building the solver ..." >> /app/oc.build.log && \
        cd app &&  \
        mkdir obj && \
        make spec && \
        rm obj/ -rf && \
        chmod 777 /app -R'

cont_img=$(buildah commit $AUTH $cont)
buildah push $AUTH $cont_img $IMG_STREAM:solver

