import os
from pathlib import Path
from typing import Any, Protocol

import httpx

from .constant import K8S_JWT_PATH
from .errors import SecretNotFoundError


class SecretProvider(Protocol):
    def get_secret(self, ref: str) -> dict[str, Any]:
        """Return secret key-value fields for `ref`"""
        ...

class OpenBaoSecretProvider:
    def __init__(
        self, base_url: str,
        k8s_role: str,
        jwt_path: Path | str = K8S_JWT_PATH,
        client: httpx.Client | None = None,
    ) -> None:
        self._token = None
        self._k8s_role = k8s_role
        self._jwt_path = Path(jwt_path)
        self._client = client or httpx.Client(
            base_url=base_url.rstrip("/"),
            timeout=10
        )

    def _get_token(self, force_refresh: bool = False) -> str:
        if self._token and not force_refresh:
            return self._token

        jwt = self._jwt_path.read_text(encoding="utf-8").strip()
        resp = self._client.post(
            "/v1/auth/kubernetes/login",
            json={"role": self._k8s_role, "jwt": jwt}
        )
        resp.raise_for_status()

        self._token = resp.json()["auth"]["client_token"]
        return self._token

    def get_secret(self, ref: str) -> dict[str, Any]:
        # Accepts format `openbao://secret/data/dev/db` or `secret/data/dev/db`

        path = ref.removeprefix("openbao://").lstrip("/")
        resp = self._client.get(
            f"/v1/{path}",
            headers={"X-Vault-Token": self._get_token()},
        )

        if resp.status_code in (401, 403):
            resp = self._client.get(
                f"/v1/{path}",
                headers={"X-Vault-Token": self._get_token(force_refresh=True)},
            )

        if resp.status_code == 404:
            raise SecretNotFoundError(f"Secret not found in OpenBao: {ref}")

        resp.raise_for_status()
        return resp.json()["data"]["data"]

def get_secret_provider() -> SecretProvider:
    # Currently only support OpenBao
    return OpenBaoSecretProvider(
        base_url=os.environ["OPENBAO_ADDR"],
        k8s_role=os.environ["OPENBAO_K8S_ROLE"]
    )
