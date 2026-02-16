# OIDC Provider Configuration Guide

## Overview

Keldris uses [OpenID Connect (OIDC)](https://openid.net/developers/how-connect-works/) for authentication. OIDC is an identity layer built on top of OAuth 2.0 that allows Keldris to verify user identity through an external identity provider (IdP) rather than managing passwords directly.

This approach provides:

- **Single sign-on (SSO)** across your infrastructure
- **Centralized user management** in your identity provider
- **Multi-factor authentication** enforced at the IdP level
- **No passwords stored** in Keldris

## Required Claims

Your OIDC provider must return the following claims in the ID token or userinfo endpoint:

| Claim   | Required | Description                        |
|---------|----------|------------------------------------|
| `sub`   | Yes      | Unique, stable user identifier     |
| `email` | Yes      | User's email address               |
| `name`  | Yes      | User's display name                |

## Redirect URL

When configuring your OIDC provider, set the redirect (callback) URL to:

```
https://your-keldris-domain/auth/callback
```

Replace `your-keldris-domain` with the actual domain where Keldris is hosted. For local development, this is typically:

```
http://localhost:8080/auth/callback
```

---

## Authentik

[Authentik](https://goauthentik.io/) is an open-source identity provider.

### 1. Create an OAuth2 Provider

1. Log in to the Authentik admin interface.
2. Navigate to **Applications** > **Providers**.
3. Click **Create** and select **OAuth2/OpenID Provider**.
4. Configure the provider:
   - **Name**: `Keldris`
   - **Authorization flow**: Select your preferred flow (e.g., `default-provider-authorization-implicit-consent`)
   - **Redirect URIs**: `https://your-keldris-domain/auth/callback`
   - **Scopes**: Ensure `openid`, `email`, and `profile` are selected
5. Click **Save**.

### 2. Create an Application

1. Navigate to **Applications** > **Applications**.
2. Click **Create**.
3. Configure:
   - **Name**: `Keldris`
   - **Slug**: `keldris`
   - **Provider**: Select the `Keldris` provider created above
4. Click **Save**.

### 3. Collect Configuration Values

From the provider settings page, note:

- **Issuer URL**: `https://your-authentik-domain/application/o/keldris/`
- **Client ID**: Shown on the provider detail page
- **Client Secret**: Shown on the provider detail page

### 4. Configure Keldris

Add the following to your `.env` file:

```env
OIDC_ISSUER_URL=https://your-authentik-domain/application/o/keldris/
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://your-keldris-domain/auth/callback
```

---

## Keycloak

[Keycloak](https://www.keycloak.org/) is an open-source identity and access management solution.

### 1. Create a Realm

1. Log in to the Keycloak admin console.
2. Click the realm dropdown in the top-left corner and select **Create Realm**.
3. Enter a **Realm name** (e.g., `keldris`) and click **Create**.

If you already have a realm you want to use, skip this step.

### 2. Create a Client

1. Navigate to **Clients** and click **Create client**.
2. Configure:
   - **Client type**: `OpenID Connect`
   - **Client ID**: `keldris`
3. Click **Next**.
4. On the **Capability config** page:
   - **Client authentication**: `On`
   - **Authentication flow**: Check `Standard flow`
5. Click **Next**.
6. On the **Login settings** page:
   - **Valid redirect URIs**: `https://your-keldris-domain/auth/callback`
   - **Web origins**: `https://your-keldris-domain`
7. Click **Save**.

### 3. Collect Configuration Values

1. Go to **Clients** > `keldris` > **Credentials** tab to find the **Client Secret**.
2. Note the following values:
   - **Issuer URL**: `https://your-keycloak-domain/realms/keldris`
   - **Client ID**: `keldris`
   - **Client Secret**: From the Credentials tab

### 4. Configure Keldris

Add the following to your `.env` file:

```env
OIDC_ISSUER_URL=https://your-keycloak-domain/realms/keldris
OIDC_CLIENT_ID=keldris
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://your-keldris-domain/auth/callback
```

---

## Okta

[Okta](https://www.okta.com/) is a cloud identity provider.

### 1. Create an OIDC Application

1. Log in to the Okta admin dashboard.
2. Navigate to **Applications** > **Applications**.
3. Click **Create App Integration**.
4. Select:
   - **Sign-in method**: `OIDC - OpenID Connect`
   - **Application type**: `Web Application`
5. Click **Next**.

### 2. Configure the Application

1. Set the following:
   - **App integration name**: `Keldris`
   - **Grant type**: Check `Authorization Code`
   - **Sign-in redirect URIs**: `https://your-keldris-domain/auth/callback`
   - **Sign-out redirect URIs**: `https://your-keldris-domain` (optional)
   - **Assignments**: Choose who can access the application
2. Click **Save**.

### 3. Collect Configuration Values

From the application's **General** tab, note:

- **Issuer URL**: `https://your-okta-domain.okta.com` (or check under **Security** > **API** > **Authorization Servers** for the full issuer URL)
- **Client ID**: Shown on the application detail page
- **Client Secret**: Shown on the application detail page

### 4. Configure Keldris

Add the following to your `.env` file:

```env
OIDC_ISSUER_URL=https://your-okta-domain.okta.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://your-keldris-domain/auth/callback
```

---

## Generic OIDC Provider

Any OIDC-compliant provider can be used with Keldris. Your provider must support the following:

### Required Endpoints

OIDC providers expose these endpoints, discoverable via the well-known configuration URL:

| Endpoint             | Description                          |
|----------------------|--------------------------------------|
| Authorization        | Where users are redirected to log in |
| Token                | Exchanges authorization code for tokens |
| Userinfo             | Returns user profile claims          |

### Finding the Issuer URL

The issuer URL is the base URL that serves the OIDC discovery document. You can verify it by fetching:

```
{issuer_url}/.well-known/openid-configuration
```

This should return a JSON document containing `authorization_endpoint`, `token_endpoint`, `userinfo_endpoint`, and other metadata.

### Testing with curl

Verify that your provider's discovery endpoint is accessible:

```bash
curl -s https://your-idp-domain/.well-known/openid-configuration | jq .
```

Expected output includes:

```json
{
  "issuer": "https://your-idp-domain",
  "authorization_endpoint": "https://your-idp-domain/authorize",
  "token_endpoint": "https://your-idp-domain/token",
  "userinfo_endpoint": "https://your-idp-domain/userinfo",
  "jwks_uri": "https://your-idp-domain/.well-known/jwks.json",
  "scopes_supported": ["openid", "email", "profile"],
  ...
}
```

Verify that `openid`, `email`, and `profile` are listed in `scopes_supported`.

### Configure Keldris

```env
OIDC_ISSUER_URL=https://your-idp-domain
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://your-keldris-domain/auth/callback
```

---

## Troubleshooting

### Common Errors

**"OIDC discovery failed" or "failed to fetch provider configuration"**

- Verify the issuer URL is correct and accessible from the Keldris server.
- Check that `/.well-known/openid-configuration` returns a valid JSON response.
- Ensure there are no firewall rules blocking outbound HTTPS from the Keldris server to the IdP.

```bash
curl -s https://your-idp-domain/.well-known/openid-configuration
```

**"Invalid redirect URI"**

- The redirect URI configured in your IdP must exactly match the `OIDC_REDIRECT_URL` in your `.env` file.
- Check for trailing slashes, protocol mismatches (`http` vs `https`), and port numbers.

**"Invalid client credentials"**

- Double-check the `OIDC_CLIENT_ID` and `OIDC_CLIENT_SECRET` values.
- Regenerate the client secret in your IdP and update the `.env` file.
- Ensure the client is not disabled in your IdP.

**"Missing required claims (email, name)"**

- Configure your IdP to include `email` and `profile` scopes.
- Some providers require explicit scope mapping. Check your provider's documentation for claim configuration.
- Verify with curl that the userinfo endpoint returns the expected claims:

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  https://your-idp-domain/userinfo
```

**"Token verification failed"**

- Ensure the Keldris server's system clock is synchronized (use NTP).
- Verify the JWKS endpoint is accessible from the Keldris server.
- Check that the ID token has not expired.

### Verifying OIDC Configuration

Run through this checklist to confirm your setup:

1. **Discovery endpoint** responds with valid JSON:
   ```bash
   curl -s $OIDC_ISSUER_URL/.well-known/openid-configuration | jq .issuer
   ```

2. **Redirect URI** in your IdP matches `OIDC_REDIRECT_URL` exactly.

3. **Client credentials** are correct and the client is enabled.

4. **Required scopes** (`openid`, `email`, `profile`) are configured on the client.

5. **Network connectivity** exists between the Keldris server and the IdP:
   ```bash
   curl -I $OIDC_ISSUER_URL/.well-known/openid-configuration
   ```

6. **Clock synchronization** on the Keldris server is accurate (token validation is time-sensitive):
   ```bash
   date -u
   ```
