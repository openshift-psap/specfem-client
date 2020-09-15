#! /bin/bash

set -e

if [[ "$SPECFEM_RELY_ON_SHARED_FS" == "true" ]]; then
    # symlinks from /app/... created in Dockerfile.mesher
    mkdir -p /mnt/shared/{DATABASES_MPI,OUTPUT_FILES}
else
    if [[ "$OMPI_COMM_WORLD_NODE_RANK" -eq 0 ]]; then
        rm -rf /app/{DATABASES_MPI,OUTPUT_FILES} # remove the symlinks
        mkdir /app/{DATABASES_MPI,OUTPUT_FILES} -p
    fi

    if [[ -z "$OMPI_COMM_WORLD_RANK" || "$OMPI_COMM_WORLD_RANK" -eq 0 ]]; then
        rm -rf /mnt/shared/DATABASES_MPI/
        mkdir /mnt/shared/DATABASES_MPI
    fi
fi

cd /app && ./bin/xmeshfem3D

if [[ -z "$OMPI_COMM_WORLD_RANK" || "$OMPI_COMM_WORLD_RANK" -eq 0 ]]; then
  cat /app/OUTPUT_FILES/output_mesher.txt | grep "buffer creation in hh:mm:ss"
  env > /mnt/shared/env.mesher
fi

[[ "$SPECFEM_RELY_ON_SHARED_FS" == "true" ]] && exit 0

if [[ -z "$OMPI_COMM_WORLD_RANK" || $OMPI_COMM_WORLD_RANK -eq 0 ]]; then
    cp /app/OUTPUT_FILES/ /mnt/shared/ -r
fi

if [[ -z "$OMPI_COMM_WORLD_RANK" ]]; then
    cp /app/DATABASES_MPI/* /mnt/shared/DATABASES_MPI/
else
    if [[ "$OMPI_COMM_WORLD_NODE_RANK" -eq 0 ]]; then
        cp /app/DATABASES_MPI/* /mnt/shared/DATABASES_MPI/
    fi
fi

echo Mesher done $OMPI_COMM_WORLD_RANK

