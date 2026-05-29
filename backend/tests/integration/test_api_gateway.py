import os

import pytest
import requests


@pytest.mark.integration
def test_generate_short_url_endpoint():
    """
    Live integration test.
    Set API_BASE_URL, for example:
    https://xxxx.execute-api.us-east-1.amazonaws.com/dev
    """
    base_url = os.environ.get("API_BASE_URL")
    if not base_url:
        pytest.skip("Set API_BASE_URL to run live integration tests.")

    response = requests.post(
        f"{base_url.rstrip('/')}/api/generate-short-url",
        json={"url": "https://example.com"},
        timeout=15,
    )

    assert response.status_code == 200
    payload = response.json()
    assert "short_url_code" in payload
