#! /bin/bash

set -e

CACHE_NAME=cache/${SPECFEM_MPI_NPROC}proc/${SPECFEM_NEX}nex/

if [ -e /mnt/shared/$CACHE_NAME/OUTPUT_FILES/values_from_mesher.h ]; then
    echo Mesher already cached $OMPI_COMM_WORLD_RANK
    exit 0
fi


if [[ "$SPECFEM_RELY_ON_SHARED_FS" == "true" ]]; then
    ln -s /mnt/shared/$CACHE_NAME/OUTPUT_FILES /app/OUTPUT_FILES
    ln -s /mnt/shared/$CACHE_NAME/DATABASES_MPI /app/DATABASES_MPI
 
    mkdir -p /mnt/shared/$CACHE_NAME/{DATABASES_MPI,OUTPUT_FILES}
else
    if [[ "$OMPI_COMM_WORLD_NODE_RANK" -eq 0 ]]; then
        mkdir /app/{DATABASES_MPI,OUTPUT_FILES} -p
    fi

    if [[ -z "$OMPI_COMM_WORLD_RANK" || "$OMPI_COMM_WORLD_RANK" -eq 0 ]]; then
        rm -rf /mnt/shared/$CACHE_NAME/
        mkdir -p /mnt/shared/$CACHE_NAME/{DATABASES_MPI,OUTPUT_FILES}
    fi
fi

cd /app && ./bin/xmeshfem3D

if [[ -z "$OMPI_COMM_WORLD_RANK" || "$OMPI_COMM_WORLD_RANK" -eq 0 ]]; then
  cat /app/OUTPUT_FILES/output_mesher.txt | grep "buffer creation in hh:mm:ss"
  env > /mnt/shared/$CACHE_NAME/env.mesher
fi

[[ "$SPECFEM_RELY_ON_SHARED_FS" == "true" ]] && exit 0

if [[ -z "$OMPI_COMM_WORLD_RANK" || $OMPI_COMM_WORLD_RANK -eq 0 ]]; then
    cp /app/OUTPUT_FILES/ /mnt/shared/$CACHE_NAME/ -r
fi

if [[ -z "$OMPI_COMM_WORLD_RANK" ]]; then
    cp /app/DATABASES_MPI/* /mnt/shared/$CACHE_NAME/DATABASES_MPI
else
    if [[ "$OMPI_COMM_WORLD_NODE_RANK" -eq 0 ]]; then
        cp /app/DATABASES_MPI/* /mnt/shared/$CACHE_NAME/DATABASES_MPI/
    fi
fi

echo Mesher done $OMPI_COMM_WORLD_RANK

