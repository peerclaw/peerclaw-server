import { fetchWithAuth } from "./client"
import type {
  ClaimToken,
  GenerateClaimTokenRequest,
  GenerateClaimTokenResponse,
} from "./types"

export function generateClaimToken(
  accessToken: string,
  params: GenerateClaimTokenRequest
): Promise<GenerateClaimTokenResponse> {
  return fetchWithAuth<GenerateClaimTokenResponse>("/claim-tokens", accessToken, {
    method: "POST",
    body: JSON.stringify(params),
  })
}

export function listClaimTokens(
  accessToken: string
): Promise<{ tokens: ClaimToken[] }> {
  return fetchWithAuth<{ tokens: ClaimToken[] }>("/claim-tokens", accessToken)
}
