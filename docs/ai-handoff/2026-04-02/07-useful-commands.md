# Useful Commands

## Checking System Health

```bash
# Status page API
curl -s "https://api.rubinot.dev/status/workers" | python3 -c "import sys,json; d=json.loads(sys.stdin.read()); [print(f'{n}: ok={j.get(\"completedLast1h\",0)} fail={j.get(\"failedLast1h\",0)}') for n,j in sorted(d['jobs'].items()) if j.get('completedLast1h',0)+j.get('failedLast1h',0)>0]"

# Queue counts
curl -s "https://api.rubinot.dev/status/queues" | python3 -m json.tool

# Experience performance history
curl -s "https://api.rubinot.dev/status/workers/highscores-fast/processors/experience/history" | python3 -c "import sys,json; [print(f'{e[\"durationMs\"]/1000:.1f}s {\"ok\" if e[\"success\"] else \"FAIL\"}') for e in json.loads(sys.stdin.read())['history'][-10:]]"
```

## Kubernetes

```bash
# Pod status
kubectl get pods -n rubinot-api --no-headers
kubectl get pods -n rubinot --no-headers

# Worker logs
kubectl logs -n rubinot-api -l app=rubinot-api-worker-heavy --tail=20 --since=5m
kubectl logs -n rubinot-api -l app=rubinot-api-worker-baseline --tail=20 --since=5m
kubectl logs -n rubinot-api -l app=rubinot-api-worker-highscore-cycle --tail=20 --since=5m

# Postgres query
kubectl exec -n rubinot-api postgres-1 -- psql -U postgres -d rubinot_api -c "SELECT ..."

# Redis
WORKER_POD=$(kubectl get pod -n rubinot-api -l app=rubinot-api-worker-heavy --no-headers -o name | head -1)
REDIS_PASS=$(kubectl exec -n rubinot-api ${WORKER_POD} -- env | grep REDIS_URL | sed 's/.*:\/\/://;s/@.*//')
kubectl exec -n rubinot-api rubinot-api-redis-5cc45584cd-hbts4 -- redis-cli -a "$REDIS_PASS" <command>

# BullMQ scheduler inspection
kubectl exec -n rubinot-api ${WORKER_POD} -- node -e "
import { Queue } from 'bullmq';
import { redisConnection } from './dist/src/jobs/queue.js';
const q = new Queue('rubinot-heavy', { connection: redisConnection });
const schedulers = await q.getJobSchedulers(0, -1, true);
for (const s of schedulers) console.log(s.key, 'every:', s.every);
await q.close(); process.exit(0);
"
```

## ClickHouse Queries

```bash
# Run via ephemeral pod
kubectl run ch-query --rm -i --restart=Never --image=curlimages/curl -n rubinot-api --timeout=20s -- \
  curl -s "http://rubinot-analytics-clickhouse-headless.rubinot-clickhouse.svc.cluster.local:8123/?query=<URL-encoded-query>"

# Example: character XP events
SELECT detected_at, category_slug, delta
FROM rubinot_analytics.highscore_change_events
WHERE character_name_normalized='prensa' AND category_slug IN ('experience','exp_today')
  AND detected_at > now() - interval 6 hour
ORDER BY detected_at
FORMAT TabSeparated

# Example: killstats change frequency
SELECT toStartOfFiveMinute(detected_at) as t, count(*) as events, countDistinct(world) as worlds
FROM rubinot_analytics.kill_statistic_change_events
WHERE race='Energetic Book' AND delta_last_day_killed != 0 AND detected_at > now() - interval 3 hour
GROUP BY t ORDER BY t DESC
FORMAT TabSeparated
```

## Grafana/Observability

```bash
# Loki logs query
# Datasource UID: loki
{namespace="rubinot-api"} |= "guilds failed" | json

# Tempo traces
# Datasource UID: tempo
# Search by service: rubinot-api or rubinot-data

# Grafana URL: grafana.cddlabs.casa (via internal gateway 10.1.120.193)
```

## Deployment

```bash
# Git identity for rubinot-lab
gh auth switch --hostname github.com --user unwashed-and-dazed

# Tag-driven deploy
git tag vX.Y.Z && git push origin vX.Y.Z

# ArgoCD refresh
kubectl patch application rubinot-lab-rubinot-api-prod -n argocd --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"normal"}}}'

# Scheduler lock clear + restart
kubectl exec -n rubinot-api <redis-pod> -- redis-cli -a "$REDIS_PASS" DEL "rubinot:scheduler:seed:lock"
kubectl delete pod -n rubinot-api -l app=rubinot-api-scheduler

# Ceph maintenance
kubectl exec -n rook-ceph <tools-pod> -- ceph osd set noout   # before node maintenance
kubectl exec -n rook-ceph <tools-pod> -- ceph osd unset noout # after node back
kubectl exec -n rook-ceph <tools-pod> -- ceph status
```
