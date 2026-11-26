import base64
import json
from pathlib import Path
from typing import Any

import pytest

from engine.core.crypto import DeterministicCrypto, apply_pii_to_cols, mask_pii
from engine.core.errors import CryptoError

VECTORS_PATH = Path(__file__).resolve().parent / "resources"/ "crypto" / "crypto-vectors.json"
VECTOR_DATA: dict[str, Any] = json.loads(VECTORS_PATH.read_text(encoding="utf-8"))
VECTORS: list[dict[str, Any]] = VECTOR_DATA["vectors"]


class _FakeSecretProvider:
    def __init__(self, secrets: dict[str, dict[str, Any]]) -> None:
        self._secrets = secrets

    def get_secret(self, ref: str) -> dict[str, Any]:
        return self._secrets[ref]


@pytest.mark.parametrize("case", VECTORS, ids=lambda c: c["name"])
def test_crypto_golden_vectors(case: dict[str, Any]) -> None:
    crypto = DeterministicCrypto.from_base64(VECTOR_DATA["key_b64"])
    plaintext = case["plaintext"]
    expected_ct = case["ciphertext_b64"]

    ct1 = crypto.encrypt(plaintext)
    ct2 = crypto.encrypt(plaintext)

    assert ct1 == expected_ct
    assert ct2 == expected_ct
    assert crypto.decrypt(ct1) == plaintext
    assert mask_pii(plaintext) == case["masked"]


def test_from_secret_ref_loads_key() -> None:
    provider = _FakeSecretProvider(
        {"openbao://secret/data/dev/pii": {"key_b64": VECTOR_DATA["key_b64"]}}
    )
    crypto = DeterministicCrypto.from_secret_ref(
        provider, "openbao://secret/data/dev/pii"
    )
    ct = crypto.encrypt("079099001234")
    assert crypto.decrypt(ct) == "079099001234"


def test_from_secret_ref_missing_field_raises() -> None:
    provider = _FakeSecretProvider({"openbao://secret/data/dev/pii": {}})
    with pytest.raises(CryptoError, match="missing string key"):
        DeterministicCrypto.from_secret_ref(provider, "openbao://secret/data/dev/pii")


@pytest.mark.parametrize(
    "bad_key",
    [b"", b"too-short", b"x" * 16, b"x" * 33],
)
def test_invalid_key_length_raises(bad_key: bytes) -> None:
    with pytest.raises(CryptoError, match="32 bytes"):
        DeterministicCrypto(bad_key)


def test_invalid_base64_key_raises() -> None:
    with pytest.raises(CryptoError, match="Invalid base64"):
        DeterministicCrypto.from_base64("not-valid-base64!!!")


def test_decrypt_rejects_tampered_wrong_key_and_corrupt_payload() -> None:
    crypto = DeterministicCrypto.from_base64(VECTOR_DATA["key_b64"])
    other = DeterministicCrypto(bytes(range(32, 64)))

    ct = crypto.encrypt("0901234567")
    assert ct is not None

    with pytest.raises(CryptoError):
        other.decrypt(ct)

    raw = bytearray(base64.b64decode(ct))
    raw[-1] ^= 0xFF
    with pytest.raises(CryptoError):
        crypto.decrypt(base64.b64encode(bytes(raw)).decode("ascii"))

    with pytest.raises(CryptoError, match="too short"):
        crypto.decrypt(base64.b64encode(b"short").decode("ascii"))

    with pytest.raises(CryptoError, match="valid base64"):
        crypto.decrypt("%%%invalid%%%")


def test_apply_pii_to_cols_inplace_and_mask_without_mutating_input() -> None:
    crypto = DeterministicCrypto.from_base64(VECTOR_DATA["key_b64"])
    original: dict[str, list[str | None]] = {
        "customer_id": ["C1", "C2", "C3"],
        "card_pan": ["4111223344556677", "0901234567", None],
    }

    inplace = apply_pii_to_cols(original, ["card_pan"], "encrypt_inplace", crypto)
    assert inplace["customer_id"] == ["C1", "C2", "C3"]
    assert "card_pan_masked" not in inplace
    assert [crypto.decrypt(v) for v in inplace["card_pan"]] == original["card_pan"]
    assert original["card_pan"][0] == "4111223344556677"

    masked = apply_pii_to_cols(original, ["card_pan"], "encrypt_and_mask", crypto)
    assert masked["card_pan"] == inplace["card_pan"]
    assert masked["card_pan_masked"] == ["411122xxxxxx6677", "******4567", None]
    assert original["card_pan"][0] == "4111223344556677"
