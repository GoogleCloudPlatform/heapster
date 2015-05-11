#!/bin/bash

set -ex
EXTRA_ARGS=""
if [ ! -z "$FLAGS" ]; then
  EXTRA_ARGS="$FLAGS"
fi

# If in Kubernetes, target the master.
if [ ! -z $KUBERNETES_SERVICE_HOST ]; then
  EXTRA_ARGS="--source=kubernetes:https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT} $EXTRA_ARGS"
fi

HEAPSTER="/usr/bin/heapster"

case $SINK in
  'influxdb') 
    # Check if in Kubernetes.
    if [ ! -z $KUBERNETES_SERVICE_HOST ]; then
    # TODO(vishh): add support for passing in user name and password.
      INFLUXDB_ADDRESS=""
      if [ ! -z $MONITORING_INFLUXDB_SERVICE_HOST ]; then
	INFLUXDB_ADDRESS="http://${MONITORING_INFLUXDB_SERVICE_HOST}:${MONITORING_INFLUXDB_SERVICE_PORT}"
      elif [ ! -z $INFLUXDB_HOST ]; then
	INFLUXDB_ADDRESS=${INFLUXDB_HOST}
      else
	echo "InfluxDB service address not specified. Exiting."
	exit 1
      fi
      $HEAPSTER --sink influxdb:$INFLUXDB_ADDRESS $EXTRA_ARGS
    elif [ ! -z $INFLUXDB_HOST ]; then
      $HEAPSTER --sink influxdb:${INFLUXDB_HOST} $EXTRA_ARGS
    else
      echo "Influxdb host invalid."
      exit 1
    fi
    ;;
  'gcm') $HEAPSTER --sink gcm $EXTRA_ARGS
    ;;
  *) $HEAPSTER $EXTRA_ARGS
    ;;
esac
