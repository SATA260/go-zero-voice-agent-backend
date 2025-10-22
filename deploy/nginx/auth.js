export default async function auth(req) {
    const token = req.headersIn['Authorization'];
    if (!token) {
        req.return(401, 'Unauthorized: No token provided');
        return;
    }

    let response;
    try {
        response = await req.subrequest(
            'http://looklook:3081/user/verify-token',
            {
                method: 'GET',
                headers: {
                    'Authorization': token
                }
            }
        );
    } catch (error) {
        req.return(500, 'Internal Server Error');
        return;
    }

    if (response.status !== 200) {
        req.return(401, 'Unauthorized: Invalid token');
        return;
    }

    let userId;
    try {
        userId = JSON.parse(response.responseBody).userId;
    } catch (error) {
        req.return(500, 'Parse Error');
        return;
    }

    req.headerOut['X-User-Id'] = userId;

    req.return(200, 'Authorized');
}