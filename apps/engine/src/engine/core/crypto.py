from __future__ import annotations

import base64
import binascii
import hashlib
import hmac
import re
from typing import Literal

from cryptography.exceptions import InvalidTag
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

from engine.core.constant import AES_256_KEY_BYTES, GCM_IV_BYTES, GCM_TAG_BYTES
from engine.core.errors import CryptoError
from engine.core.secrets import SecretProvider


class DeterministicCrypto:
    """Deterministic AES-256-GCM encryption and display masking for PII columns"""

    def __init__(self, key: bytes) -> None:
        if len(key) != AES_256_KEY_BYTES:
            raise CryptoError(f"AES-256 key must be {AES_256_KEY_BYTES} bytes, got {len(key)}")

        self._key = key
        self._aesgcm = AESGCM(key)

    @classmethod
    def from_base64(cls, key_b64: str) -> DeterministicCrypto:
        try:
            key = base64.b64decode(key_b64, validate=True)
        except (binascii.Error, ValueError) as exc:
            raise CryptoError("Invalid base64 AES key") from exc
        return cls(key)

    @classmethod
    def from_secret_ref(
        cls,
        provider: SecretProvider,
        key_ref: str,
        key: str = "key_b64"
    ) -> DeterministicCrypto:
        secret = provider.get_secret(key_ref)

        raw = secret.get(key)
        if not isinstance(raw, str) or not raw:
            raise CryptoError(f"Secret '{key_ref}' missing string key")

        return cls.from_base64(raw)


    def _derive_iv(self, plain_bytes: bytes) -> bytes:
        return hmac.new(self._key, plain_bytes, hashlib.sha256).digest()[:GCM_IV_BYTES]

    def encrypt(self, value: str | None) -> str | None:
        if value is None:
            return None

        plain_bytes = value.encode("utf-8")
        iv = self._derive_iv(plain_bytes)
        ct_and_tag = self._aesgcm.encrypt(iv, plain_bytes, None)

        return base64.b64encode(iv + ct_and_tag).decode("ascii")

    def decrypt(self, token: str | None) -> str | None:
        if token is None:
            return None

        if token == "":
            return ""

        try:
            raw = base64.b64decode(token, validate=True)
        except (binascii.Error, ValueError) as exc:
            raise CryptoError("Ciphertext is not valid base64") from exc

        if len(raw) < GCM_IV_BYTES + GCM_TAG_BYTES:
            raise CryptoError("Ciphertext payload is too short")

        iv, ct_and_tag = raw[:GCM_IV_BYTES], raw[GCM_IV_BYTES:]

        try:
            plain_bytes = self._aesgcm.decrypt(iv, ct_and_tag, None)
            return plain_bytes.decode("utf-8")
        except (InvalidTag, UnicodeDecodeError) as exc:
            raise CryptoError("Failed to decrypt ciphertext") from exc

_CARD_PAN_RE = re.compile(r"^\d{16,19}$")

def mask_pii(value: str | None) -> str | None:
    if not value:
        return None

    if _CARD_PAN_RE.match(value):
        return f"{value[:6]}{'x' * (len(value) - 10)}{value[-4:]}"

    if len(value) <= 4:
        return '*' * len(value)

    return f"{'*' * (len(value) - 4)}{value[-4:]}"

def apply_pii_to_cols(
    columns: dict[str, list[str | None]],
    pii_columns: list[str],
    mode: Literal["encrypt_inplace", "encrypt_and_mask"],
    crypto: DeterministicCrypto
) -> dict[str, list[str | None]]:

    out = dict(columns)
    for col in pii_columns:
        values = out[col]
        if mode == "encrypt_and_mask":
            out[f"{col}_masked"] = [mask_pii(v) for v in values]
        out[col] = [crypto.encrypt(v) for v in values]
    return out
