import { fetchWithAuth } from "./client"
import type { ClaimToken, GenerateClaimTokenResponse } from "./types"

export function generateClaimToken(
  accessToken: string
): Promise<GenerateClaimTokenResponse> {
  return fetchWithAuth<GenerateClaimTokenResponse>("/claim-tokens", accessToken, {
    method: "POST",
  })
}

export function listClaimTokens(
  accessToken: string
): Promise<{ tokens: ClaimToken[] }> {
  return fetchWithAuth<{ tokens: ClaimToken[] }>("/claim-tokens", accessToken)
}
