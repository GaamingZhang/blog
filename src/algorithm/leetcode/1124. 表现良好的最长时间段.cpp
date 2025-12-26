class Solution
{
public:
    int longestWPI(vector<int> &hours)
    {
        int n = hours.size(), ans = 0;
        vector<int> preSum(n + 1, 0);
        stack<int> st;
        st.push(0);
        for (int i = 0; i < n; ++i)
        {
            preSum[i + 1] = preSum[i] + (hours[i] > 8 ? 1 : -1);
            if (preSum[i + 1] < preSum[st.top()])
                st.push(i + 1);
        }
        for (int i = n; i > 0; --i)
        {
            while (!st.empty() && preSum[i] > preSum[st.top()])
            {
                ans = max(ans, i - st.top());
                st.pop();
            }
        }
        return ans;
    }
};