import json
from collections.abc import Callable
from pathlib import Path

import httpx
import pytest

from engine.core.errors import SecretNotFoundError
from engine.core.secrets import OpenBaoSecretProvider, get_secret_provider


def _build_provider(
    tmp_path: Path,
    handler: Callable[[httpx.Request], httpx.Response],
    *,
    jwt_token: str = "fake-k8s-jwt",
) -> OpenBaoSecretProvider:
    jwt_file = tmp_path / "token"
    jwt_file.write_text(f"  {jwt_token}\n", encoding="utf-8")
    client = httpx.Client(
        base_url="http://openbao.local:8200",
        transport=httpx.MockTransport(handler),
    )
    return OpenBaoSecretProvider(
        base_url="http://openbao.local:8200",
        k8s_role="engine-worker",
        jwt_path=jwt_file,
        client=client,
    )


def test_openbao_k8s_login_and_caches_token(tmp_path: Path) -> None:
    login_calls = 0
    seen_paths: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal login_calls
        if request.url.path == "/v1/auth/kubernetes/login":
            login_calls += 1
            body = json.loads(request.content.decode("utf-8"))
            assert body == {"role": "engine-worker", "jwt": "fake-k8s-jwt"}
            return httpx.Response(200, json={"auth": {"client_token": "bao-tok-1"}})

        seen_paths.append(request.url.path)
        assert request.headers["X-Vault-Token"] == "bao-tok-1"
        return httpx.Response(
            200,
            json={"data": {"data": {"host": "pg.local", "port": 5432}}},
        )

    provider = _build_provider(tmp_path, handler)
    first = provider.get_secret("openbao://secret/data/dev/db")
    second = provider.get_secret("secret/data/dev/api")

    assert first == {"host": "pg.local", "port": 5432}
    assert second == {"host": "pg.local", "port": 5432}
    assert login_calls == 1
    assert seen_paths == ["/v1/secret/data/dev/db", "/v1/secret/data/dev/api"]


@pytest.mark.parametrize("status_code", [401, 403])
def test_openbao_refreshes_token_on_auth_error(
    tmp_path: Path, status_code: int
) -> None:
    login_calls = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal login_calls
        if request.url.path == "/v1/auth/kubernetes/login":
            login_calls += 1
            return httpx.Response(
                200, json={"auth": {"client_token": f"bao-tok-{login_calls}"}}
            )

        if request.headers.get("X-Vault-Token") == "bao-tok-1":
            return httpx.Response(status_code)

        assert request.headers["X-Vault-Token"] == "bao-tok-2"
        return httpx.Response(200, json={"data": {"data": {"key_b64": "YWJj"}}})

    provider = _build_provider(tmp_path, handler)
    assert provider.get_secret("openbao://secret/data/dev/key") == {"key_b64": "YWJj"}
    assert login_calls == 2


def test_openbao_404_raises_secret_not_found(tmp_path: Path) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/v1/auth/kubernetes/login":
            return httpx.Response(200, json={"auth": {"client_token": "bao-tok"}})
        return httpx.Response(404)

    provider = _build_provider(tmp_path, handler)
    with pytest.raises(SecretNotFoundError, match="secret/data/missing"):
        provider.get_secret("openbao://secret/data/missing")


def test_openbao_500_raises_http_status_error(tmp_path: Path) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/v1/auth/kubernetes/login":
            return httpx.Response(200, json={"auth": {"client_token": "bao-tok"}})
        return httpx.Response(500)

    provider = _build_provider(tmp_path, handler)
    with pytest.raises(httpx.HTTPStatusError):
        provider.get_secret("openbao://secret/data/dev/db")


def test_openbao_strips_extra_leading_slashes(tmp_path: Path) -> None:
    seen_paths: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/v1/auth/kubernetes/login":
            return httpx.Response(200, json={"auth": {"client_token": "bao-tok"}})
        seen_paths.append(request.url.path)
        return httpx.Response(200, json={"data": {"data": {"ok": True}}})

    provider = _build_provider(tmp_path, handler)
    assert provider.get_secret("openbao:///secret/data/dev/db") == {"ok": True}
    assert seen_paths == ["/v1/secret/data/dev/db"]


def test_get_secret_provider_reads_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("OPENBAO_ADDR", "http://openbao.infra.svc:8200")
    monkeypatch.setenv("OPENBAO_K8S_ROLE", "engine-worker")

    provider = get_secret_provider()
    assert isinstance(provider, OpenBaoSecretProvider)


def test_get_secret_provider_missing_env_raises(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("OPENBAO_ADDR", raising=False)
    monkeypatch.delenv("OPENBAO_K8S_ROLE", raising=False)

    with pytest.raises(KeyError):
        get_secret_provider()
