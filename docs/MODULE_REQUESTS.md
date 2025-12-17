# Module Request System

## Overview

The CLIPilot Registry includes a **Module Request System** that automatically tracks queries from users when no matching module is found. This creates a feedback loop where the community can see what modules are most needed and prioritize development accordingly.

## How It Works

### For Users

When you search for a module that doesn't exist:

```bash
$ clipilot search "setup kubernetes cluster"
No modules found matching your query.
Try different keywords or check installed modules with: modules list

💡 Your request has been submitted to help us improve CLIPilot.
   Check https://clipilot.themobileprof.com for new modules!
```

**What happens:**
1. Your query is automatically sent to the registry
2. The registry stores your request with context (OS, Termux status)
3. Contributors can see what modules are being requested
4. You get notified that your feedback was received

**Privacy:**
- Only the query text and basic context (OS type, Termux yes/no) are sent
- IP address is logged for duplicate detection but not displayed publicly
- No personal information is collected

### For Contributors

Contributors with admin access can view module requests at:
**https://clipilot.themobileprof.com/module-requests**

**Features:**
- View all pending requests
- Filter by status: pending, in_progress, completed, duplicate
- See request frequency and patterns
- Mark requests as fulfilled when modules are created
- Add notes and track duplicate requests

## API Endpoints

### Submit Module Request (Public)

**POST** `/api/module-request`

Submit a request when no matching module is found.

**Request Body:**
```json
{
  "query": "how to setup kubernetes",
  "user_context": "os=linux, termux=false"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Thank you! Your request has been received...",
  "request_id": 42,
  "status": "pending"
}
```

### List Module Requests (Admin Only)

**GET** `/module-requests?status=pending`

View module requests with optional status filter.

**Query Parameters:**
- `status` - Filter by status: `pending`, `in_progress`, `completed`, `duplicate`, `all`

### Update Module Request (Admin Only)

**PUT** `/api/module-request/:id`

Update a module request's status, notes, or fulfillment info.

**Request Body:**
```json
{
  "status": "completed",
  "notes": "Created where_is_web_root module to fulfill this request",
  "fulfilled_by_module": "where_is_web_root"
}
```

## Database Schema

```sql
CREATE TABLE module_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query TEXT NOT NULL,
    user_context TEXT,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending',
    duplicate_of INTEGER,
    notes TEXT,
    fulfilled_by_module TEXT,
    FOREIGN KEY (duplicate_of) REFERENCES module_requests(id)
);
```

## Admin Workflow

### 1. Review Pending Requests

Visit `/module-requests` to see all pending requests sorted by date.

### 2. Identify Patterns

Look for:
- **Frequent queries** - Multiple users asking for the same thing
- **Clear needs** - Specific, actionable requests
- **Gaps** - Areas where CLIPilot has no coverage

### 3. Create Modules

When creating a module to fulfill a request:
1. Develop and test the module
2. Upload to the registry
3. Mark the request as "completed"
4. Add the module name in "fulfilled_by_module"

### 4. Mark Duplicates

If multiple requests ask for the same thing:
1. Keep the oldest/clearest request
2. Mark others as "duplicate"
3. Set `duplicate_of` to the original request ID

### 5. Track Progress

Use "in_progress" status for requests you're actively working on.

## Benefits

**For Users:**
- ✅ Seamless experience - happens automatically
- ✅ Feel heard - feedback is tracked and visible
- ✅ See progress - check registry for new modules

**For Contributors:**
- ✅ Data-driven priorities - build what users need
- ✅ Community insight - understand pain points
- ✅ Impact tracking - see fulfilled requests

**For the Project:**
- ✅ User engagement - community involvement
- ✅ Module roadmap - natural prioritization
- ✅ Quality feedback - real-world use cases

## Future Enhancements

The module request endpoint is designed to be extensible:

### Phase 1: Data Collection (Current)
- Collect queries from users
- Display to admins
- Manual fulfillment

### Phase 2: LLM Integration (Planned)
```json
{
  "success": true,
  "message": "Analyzing your request...",
  "suggestion": "Generated module suggestion from LLM",
  "confidence": 0.85
}
```

The same endpoint will:
- Query an external LLM (OpenAI, Anthropic, etc.)
- Generate module suggestions in real-time
- Still log the request for tracking
- Return AI-generated responses to users

### Phase 3: Auto-Generation (Future)
- LLM generates complete module YAML
- Admin reviews and approves
- Automatic deployment to registry

## Example Admin View

```
Module Requests (Pending: 15)

#42 | Pending | Jan 15, 2025
Query: "how to setup kubernetes cluster"
Context: os=linux, termux=false
[Mark In Progress] [Mark Completed] [Mark Duplicate] [Add Notes]

#41 | In Progress | Jan 14, 2025
Query: "install and configure redis"
Context: os=linux, termux=false
Notes: Working on redis_setup module
[Mark Completed] [Add Notes]

#40 | Completed | Jan 13, 2025
Query: "find nginx config file"
Context: os=linux, termux=true
Fulfilled by: where_is_config
```

## Privacy & Security

**Data Collected:**
- Query text (user input)
- OS type (linux/darwin/etc)
- Termux flag (boolean)
- IP address (for deduplication only)
- User agent (client version)

**Data NOT Collected:**
- User names or identifiers
- File paths or system details
- Command history
- Any personal information

**Access Control:**
- Public: Submit requests
- Admin only: View and manage requests
- Anonymous: No login required to submit

**Rate Limiting:**
- Consider implementing per-IP limits to prevent abuse
- Current: No limit (trust-based)

## Contributing

To improve the module request system:

1. **Enhanced filtering** - Add search, date ranges, sorting
2. **Analytics dashboard** - Most requested keywords, trends
3. **Email notifications** - Alert admins of new high-value requests
4. **Public voting** - Let users upvote existing requests
5. **LLM integration** - Generate responses automatically

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.
