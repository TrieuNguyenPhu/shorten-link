"""Offline planning estimate, not an AWS bill or a spending cap.

Uses the checked-in Singapore/Asia price snapshot. No AWS credentials or API
calls. Free tiers, credits, domain registration, tax and v1 resources excluded.
Run from any directory: python3 /absolute/path/to/estimate_serverless.py
"""

from decimal import Decimal, ROUND_HALF_UP
import json
from pathlib import Path


D = Decimal
PRICE_FILE = Path(__file__).with_name('2026-11-12-singapore-rates.json')
CENT = D('0.01')


def money(value):
    return value.quantize(CENT, rounding=ROUND_HALF_UP)


def estimate(clicks, rates):
    n = D(clicks)
    if n <= 0:
        raise ValueError('clicks must be positive')
    management = n * D('0.5')  # Includes authentication, listing and now polling.
    hours = D('730')
    minutes = hours * 60
    # Minute, five-minute, hourly, daily, weekly jobs. Conservative rounding.
    jobs = minutes + minutes / 5 + hours + D(31) + D(5)
    scale = n / D(100000)
    http_calls = n + management
    lambda_calls = http_calls + n + jobs  # No worker batching discount assumed.
    lambda_gb_seconds = (
        http_calls * D('0.5') * D('0.2')  # API: 512 MiB, average 200 ms.
        + n * D('0.5') * D('0.1')         # Enrichment: 512 MiB, 100 ms/event.
        + jobs * D('0.5') * D(2)          # Jobs: 512 MiB, average 2 seconds.
    )
    # Billing-unit envelopes, not API-operation counts. Raw rows <=2 KiB,
    # transactions, one raw time index, live counters and amortized rollups.
    # Jobs/read models exceeding this envelope require a revised estimate.
    writes = 12 * n + D('0.1') * management + 3 * jobs
    reads = 4 * n + 4 * management + 8 * jobs
    ddb_gb = D('0.5') + D('2.5') * n / D(1000000)
    logs_gb = D('0.25') + D('0.25') * scale
    values = {
        'Lambda ARM': lambda_gb_seconds * rates['lambda_gb_second']
        + lambda_calls * rates['lambda_request'],
        'HTTP API': http_calls * rates['http_api_request'],
        'DynamoDB + metadata PITR': writes * rates['ddb_write_unit']
        + reads * rates['ddb_read_unit']
        + ddb_gb * rates['ddb_storage_gb_month']
        + D('0.1') * rates['ddb_pitr_gb_month'],
        'SQS incl. polling allowance': (3 * n + 3 * jobs + D(1500000))
        * rates['sqs_request'],
        'CloudFront Asia': (http_calls + D(50000) * scale)
        * rates['cloudfront_https_request']
        + 3 * scale * rates['cloudfront_outbound_gb']
        + D('0.1') * scale * rates['cloudfront_to_origin_gb'],
        'S3 storage/requests': (D(2) + n / D(1000000))
        * rates['s3_storage_gb_month']
        + D(5000) * rates['s3_put_request']
        + D(50000) * rates['s3_get_request'],
        'Logs + 3 custom metrics + 5 alarms': logs_gb * rates['logs_ingest_gb']
        + logs_gb * rates['logs_storage_gb_month']
        + 3 * rates['custom_metric_month']
        + 5 * rates['standard_alarm_month'],
        # An explicit planning allowance, not a quoted price dimension.
        'Scheduler/SSM/SNS/DNS/API extras allowance': D(1),
    }
    costs = {name: money(value) for name, value in values.items()}
    subtotal = sum(costs.values(), D(0))
    reserve = money(subtotal * D('0.30'))
    return {
        'clicks': int(n),
        'management_and_polling_calls': int(management),
        'scheduled_jobs': int(jobs),
        'http_calls': int(http_calls),
        'lambda_calls': int(lambda_calls),
        'ddb_write_units': str(writes),
        'ddb_read_units': str(reads),
        'data_transfer_gb': str(3 * scale),
        'costs': {name: str(value) for name, value in costs.items()},
        'subtotal_usd': str(subtotal),
        'reserve_30_percent_usd': str(reserve),
        'planning_total_usd': str(subtotal + reserve),
        'within_20_usd_scenario': subtotal + reserve < D(20),
    }


def main():
    snapshot = json.loads(PRICE_FILE.read_text())
    assert snapshot['region'] == 'ap-southeast-1'
    rates = {key: D(item['usd']) for key, item in snapshot['rates'].items()}
    scenarios = [estimate(n, rates) for n in (100000, 300000, 1000000)]
    for result in scenarios:
        assert sum(map(D, result['costs'].values())) == D(result['subtotal_usd'])
        assert D(result['planning_total_usd']) == (
            D(result['subtotal_usd']) + D(result['reserve_30_percent_usd'])
        )
    print(json.dumps({
        'price_snapshot': PRICE_FILE.name,
        'assumptions': snapshot['policy'],
        'warning': 'Modeled scenarios only. Buffers do not enforce a hard cap; '
        'traffic, resource use and measured durations can exceed assumptions.',
        'scenarios': scenarios,
    }, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
