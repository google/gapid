#!/bin/sh
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

set -eu

cd "$(dirname $0)"

if ! (which bq) > /dev/null; then
  echo "Please install 'bq' from gcloud SDK"
  echo "  https://cloud.google.com/sdk/install"
  exit 1
fi

if ! (which bqschemaupdater) > /dev/null; then
  echo "Please install 'bqschemaupdater' from Chrome's infra.git"
  echo "  Checkout infra.git then run: eval \`./go/env.py\`"
  exit 1
fi

if [ $# != 1 ]; then
  echo "usage: setup_bigquery.sh <instanceid>"
  echo ""
  echo "Pass one argument which is the instance name"
  exit 1
fi

APPID=$1

echo "- Make sure the BigQuery API is enabled for the project:"
# It is enabled by default for new projects, but it wasn't for older projects.
gcloud services enable --project ${APPID} bigquery-json.googleapis.com


# TODO(maruel): The stock role "roles/bigquery.dataEditor" grants too much
# rights. Create a new custom role with only access
# "bigquery.tables.updateData".
#echo "- Create a BQ write-only role account:"
# https://cloud.google.com/iam/docs/understanding-custom-roles


# https://cloud.google.com/iam/docs/granting-roles-to-service-accounts
# https://cloud.google.com/bigquery/docs/access-control
echo "- Grant access to the AppEngine app to the role account:"
gcloud projects add-iam-policy-binding ${APPID} \
    --member serviceAccount:${APPID}@appspot.gserviceaccount.com \
    --role roles/bigquery.dataEditor


echo "- Create the dataset:"
echo ""
echo "  Warning: On first 'bq' invocation, it'll try to find out default"
echo "    credentials and will ask to select a default app; just press enter to"
echo "    not select a default."

if ! (bq --location=US mk --dataset \
  --description 'Swarming statistics' ${APPID}:swarming); then
  echo ""
  echo "Dataset creation failed. Assuming the dataset already exists. At worst"
  echo "the following command will fail."
fi


echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
cd proto/api
# 1.5 years = (365 + 182) * 24 hours.
if ! (bqschemaupdater -force \
    -partitioning-expiration 13128h \
    -message swarming.v1.BotEvent \
    -table ${APPID}.swarming.bot_events \
    -partitioning-field event_time); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.bot_events"
  echo ""
  echo "and run this script again."
  exit 1
fi
if ! (bqschemaupdater -force \
    -partitioning-expiration 13128h \
    -message swarming.v1.TaskRequest \
    -table ${APPID}.swarming.task_requests \
    -partitioning-field create_time); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.task_requests"
  echo ""
  echo "and run this script again."
  exit 1
fi
if ! (bqschemaupdater -force \
    -partitioning-expiration 13128h \
    -message swarming.v1.TaskResult \
    -table ${APPID}.swarming.task_results_run \
    -partitioning-field end_time); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.task_results_run"
  echo ""
  echo "and run this script again."
  exit 1
fi
if ! (bqschemaupdater -force \
    -partitioning-expiration 13128h \
    -message swarming.v1.TaskResult \
    -table ${APPID}.swarming.task_results_summary \
    -partitioning-field end_time); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.task_results_summary"
  echo ""
  echo "and run this script again."
  exit 1
fi
cd -

echo "- Create BigQuery views:"
echo ""
echo " - swarming.bot_events_delta"
QUERY='SELECT
  DATE(event_time) AS day,
  TIMESTAMP_TRUNC(event_time, SECOND) AS time,
  ROUND(
    TIMESTAMP_DIFF(
      event_time,
      CAST(
        LEAD(event_time) OVER(PARTITION BY bot.bot_id ORDER BY event_time DESC)
        AS TIMESTAMP),
      MICROSECOND)*0.000001,
    3) AS since_last,
  ROUND(
    TIMESTAMP_DIFF(
      CAST(
        LAG(event_time) OVER(PARTITION BY bot.bot_id ORDER BY event_time DESC)
        AS TIMESTAMP),
      event_time,
      MICROSECOND)*0.000001,
    3) AS until_next,
  ARRAY(
    SELECT FORMAT("%s:%s", d.key, v) FROM e.bot.dimensions AS d, d.values AS v)
      AS dimensions_flat,
  *
FROM `'${APPID}'.swarming.bot_events` AS e
WHERE event_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 22 DAY)
ORDER BY event_time DESC'
DESC="Augmented view of last 3 weeks with deltas between events and \
dimensions_flat"
if !(bq mk --use_legacy_sql=false --view "${QUERY}" \
  --description "{$DESC}" \
  --project_id ${APPID} swarming.bot_events_delta); then
  echo ""
  echo "The view already exists. You can delete it with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.bot_events_delta"
  echo ""
  echo "and run this script again."
  # Don't fail here.
fi

echo ""
echo " - swarming.task_requests_flat"
QUERY='SELECT
  DATE(create_time) AS day,
  TIMESTAMP_TRUNC(create_time, SECOND) AS time,
  ARRAY(
    SELECT DISTINCT FORMAT("%s:%s", dimensions.key, value)
    FROM
      UNNEST(r.task_slices) AS slice,
      UNNEST(slice.properties.dimensions) AS dimensions,
      UNNEST(dimensions.values) AS value)
    AS dimensions_flat,
  *
FROM `'${APPID}'.swarming.task_requests` AS r
WHERE create_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 22 DAY)
ORDER BY create_time DESC'
DESC="Augmented view of last 3 weeks with dimensions_flat"
if !(bq mk --use_legacy_sql=false --view "${QUERY}" \
  --description "{$DESC}" \
  --project_id ${APPID} swarming.task_requests_flat); then
  echo ""
  echo "The view already exists. You can delete it with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.task_requests_flat"
  echo ""
  echo "and run this script again."
  # Don't fail here.
fi

echo ""
echo " - swarming.task_results_run_flat"
QUERY='SELECT
  DATE(end_time) AS day,
  TIMESTAMP_TRUNC(end_time, SECOND) AS time,
  ARRAY(
    SELECT DISTINCT FORMAT("%s:%s", dimensions.key, value)
    FROM
      UNNEST(r.request.task_slices[
        OFFSET(r.current_task_slice)].properties.dimensions) AS dimensions,
      UNNEST(dimensions.values) AS value)
    AS dimensions_flat,
  *
FROM `'${APPID}'.swarming.task_results_run` AS r
WHERE end_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 22 DAY)
ORDER BY end_time DESC'
DESC="Augmented view of last 3 weeks with dimensions_flat"
if !(bq mk --use_legacy_sql=false --view "${QUERY}" \
  --description "{$DESC}" \
  --project_id ${APPID} swarming.task_results_run_flat); then
  echo ""
  echo "The view already exists. You can delete it with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.task_results_run_flat"
  echo ""
  echo "and run this script again."
  # Don't fail here.
fi

echo ""
echo " - swarming.task_results_summary_flat"
QUERY='SELECT
  DATE(end_time) AS day,
  TIMESTAMP_TRUNC(end_time, SECOND) AS time,
  ARRAY(
    SELECT DISTINCT FORMAT("%s:%s", dimensions.key, value)
    FROM
      UNNEST(r.request.task_slices[
        OFFSET(r.current_task_slice)].properties.dimensions) AS dimensions,
      UNNEST(dimensions.values) AS value)
    AS dimensions_flat,
  *
FROM `'${APPID}'.swarming.task_results_summary` AS r
WHERE end_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 22 DAY)
ORDER BY end_time DESC'
DESC="Augmented view of last 3 weeks with dimensions_flat"
if !(bq mk --use_legacy_sql=false --view "${QUERY}" \
  --description "{$DESC}" \
  --project_id ${APPID} swarming.task_results_summary_flat); then
  echo ""
  echo "The view already exists. You can delete it with:"
  echo ""
  echo "  bq rm ${APPID}:swarming.task_results_summary_flat"
  echo ""
  echo "and run this script again."
  # Don't fail here.
fi
