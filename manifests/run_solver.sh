#! /bin/bash

set -e

cd app

./bin/xspecfem3D

if [[ -z "$OMPI_COMM_WORLD_RANK" || $OMPI_COMM_WORLD_RANK -eq 0 ]]; then
  cp oc.build.log /mnt/shared/
  env > /mnt/shared/env.solver
  cat OUTPUT_FILES/output_solver.txt | grep "Total elapsed time in seconds"
fi
