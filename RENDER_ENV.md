# Render Environment Variables Setup

## Required Environment Variables

### Backend Service (`testai-backend`)

#### Gemini API Keys (Pool-based)
- `GEMINI_API_KEYS` - Comma-separated Gemini API keys for load balancing
  - Format: "key1,key2,key3,key4,key5,key6"
  - Automatically used in round-robin fashion
  - Primary redundancy mechanism

#### Gemini Fallback
- `GEMINI_API_KEY_FALLBACK` - Single fallback key if pool fails

#### Search API Keys (Pool-based)
- `SERPAPI_KEYS` - Comma-separated SerpAPI keys
  - Format: "key1,key2"
  - Automatically switches on rate limit/expiration
- `TAVILY_KEYS` - Comma-separated Tavily keys
  - Format: "key1,key2"
  - Load balanced across requests

#### Optional LLM API Keys
- `OPENROUTER_KEYS` - Comma-separated OpenRouter keys
  - Format: "key1,key2"
  - Alternative LLM provider if Gemini fails

#### Configuration
- `GEMINI_MODEL` - Default: `gemini-3.1-flash-lite`
- `REDIS_URL` - Auto-populated from Redis service
- `NODE_ENV` - `production`
- `LOG_LEVEL` - `info`
- `PORT` - Auto-assigned by Render (app uses 8080 internally)

### Frontend Service (`testai-frontend`)
- `NODE_ENV` - `production`

## How to Set Up on Render

1. **Login to Render Dashboard**
   - Go to https://render.com/dashboard

2. **Create New Service from Blueprint**
   - Click "Add +" → "New Web Service"
   - Select "Deploy from Git Repository"
   - Choose your repository
   - Use the `render.yaml` blueprint

3. **Set Environment Variables**
   - For each service, go to "Settings" → "Environment"
   - Add all required API keys as secrets
   - Make sure to mark them as "Sync: false" for security

4. **API Key Sources**
   
   **Gemini API Keys:**
   - Go to https://makersuite.google.com/app/apikey
   - Create multiple API keys for redundancy
   
   **SerpAPI Keys:**
   - Go to https://serpapi.com/
   - Sign up and get your API key
   
   **Tavily API Keys:**
   - Go to https://tavily.com/
   - Sign up and get your API key
   
   **OpenRouter API Keys:**
   - Go to https://openrouter.ai/
   - Sign up and get your API key

5. **Service Configuration**
   
   **Backend Service:**
   - Plan: Free (or upgrade when needed)
   - Region: Singapore (closest to VN)
   - Health Check: `/health` (verify in Go code)
   - Build Command: Uses Dockerfile
   - Start Command: `./gemini-cli --server`
   
   **Redis Service:**
   - Plan: Free
   - Region: Singapore
   
   **Frontend Service:**
   - Plan: Free
   - Region: Singapore
   - Build Command: `cd frontend && npm ci && npm run build`
   - Start Command: `nginx -g 'daemon off;'`

## Testing Locally

Before deploying, test locally:

```bash
# Start services
docker-compose up -d

# Check health
curl http://localhost:8080/health
curl http://localhost:3000

# View logs
docker-compose logs -f backend
docker-compose logs -f frontend
```

## Monitoring & Logging

- **Health Checks**: Both services have automatic health checks
- **Logs**: Available in Render dashboard and via Docker locally
- **Metrics**: Render provides basic metrics for CPU, memory, requests
- **Alerts**: Set up alerts in Render dashboard for failed health checks

## Cost Estimation

- **Free Tier**: 1 CPU, 512MB-1GB RAM, 1GB storage per service
- **Estimated Monthly Cost**: ~$0 (within free limits)
- **Upgrade Cost**: 
  - Hobby: $7/month per service
  - Starter: $20/month per service

## Security Notes

1. Never commit API keys to git
2. Use `sync: false` for all sensitive environment variables
3. Enable service-to-service authentication if needed
4. Regularly rotate API keys
5. Monitor for abnormal usage patterns

## Troubleshooting

Common issues:

1. **Build Failures**: Check Dockerfile paths and dependencies
2. **Runtime Errors**: Verify environment variables and health check paths
3. **Connection Issues**: Ensure Redis URL is correctly configured
4. **Performance Issues**: Monitor CPU/memory usage and consider scaling