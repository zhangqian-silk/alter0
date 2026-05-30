export function hashSessionIDShort(value: string): string {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(16).padStart(8, "0").slice(0, 8);
}

export function isSessionShortHash(value: string): boolean {
  return /^[0-9a-f]{8}$/i.test(value.trim());
}

export function sessionRouteToken(value: string): string {
  const normalized = value.trim();
  if (!normalized) {
    return "";
  }
  return isSessionShortHash(normalized) ? normalized.toLowerCase() : hashSessionIDShort(normalized);
}

export function sessionIDMatchesReference(sessionID: string, reference: string): boolean {
  const normalizedID = sessionID.trim();
  const normalizedReference = reference.trim();
  if (!normalizedID || !normalizedReference) {
    return false;
  }
  return normalizedID === normalizedReference || hashSessionIDShort(normalizedID) === normalizedReference.toLowerCase();
}

export function resolveSessionIDReference<T extends { id: string }>(
  sessions: T[],
  reference: string,
): string {
  const normalizedReference = reference.trim();
  if (!normalizedReference) {
    return "";
  }
  return sessions.find((session) => sessionIDMatchesReference(session.id, normalizedReference))?.id || "";
}
