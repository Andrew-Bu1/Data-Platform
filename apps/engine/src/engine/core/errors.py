class SecretNotFoundError(Exception):
    """Raise when a secret reference does not exist."""

class CryptoError(Exception):
    """Raised when encryption, decryption, or key validation fails."""
