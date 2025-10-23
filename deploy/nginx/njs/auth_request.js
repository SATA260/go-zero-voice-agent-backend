async function authorize(r) {
    var token = r.headersIn.Authorization;

    if (!token) {
        r.error("No token");
        r.return(401);
        return;
    }

    // if (r.method != 'GET') {
    //     r.error(`Unsupported method: ${r.method}`);
    //     r.return(401);
    //     return;
    // }

    // var args = r.variables.args;

    // var h = crypto.createHmac('sha1', process.env.SECRET_KEY);

    // h.update(r.uri).update(args ? args : "");

    // var req_sig = h.digest("base64");

    // if (req_sig != token) {
    //     r.error(`Invalid token: ${req_sig}\n`);
    //     r.return(401);
    //     return;
    // }

    try {
        const reply = await r.subrequest('/usercenter/v1/user/auth', {
            method: 'GET',
            headers: { 'Authorization': token }
        });

        if (reply.status === 200) {
            const body = reply.responseBody || reply.responseText || reply.body || '';

            // --- 新增调试日志 ---
            r.error(`Auth service returned status 200 with body: ${body}`);

            let data = null;
            try {
                data = body ? JSON.parse(body) : null;
            } catch (e) {
                r.error('Auth service returned non-JSON body');
                r.return(500, 'Auth service error');
                return;
            }

            // --- 新增调试日志 ---
            r.error(`Parsed data: ${JSON.stringify(data)}`);

            // 根据后端返回的字段名尝试提取用户 ID
            const userId = data && (data.userId || data.UserId || data.id || data.ID);

            // --- 新增调试日志 ---
            r.error(`Extracted userId: ${userId}`);

            if (userId) {
                // 将 userId 添加到后续请求头，后端可以读取 X-User-Id
                r.headersOut['X-User-Id'] = String(userId);
            }

            r.return(200);
            return;
        } else {
            r.error(`Auth failed: status ${reply.status}`);
            r.return(401);
            return;
        }
    } catch (e) {
        r.error(`Subrequest error: ${e.message}`);
        r.return(500, 'Auth service error');
        return;
    }

}

export default { authorize }