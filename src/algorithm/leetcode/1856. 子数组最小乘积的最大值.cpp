class Solution
{
public:
    int maxSumMinProduct(vector<int> &nums)
    {
        const int MOD = 1e9 + 7;
        vector<long long> preSum(nums.size() + 1, 0LL);
        for (int i = 0; i < nums.size(); ++i)
        {
            preSum[i + 1] = preSum[i] + nums[i];
        }
        vector<int> left(nums.size(), -1), right(nums.size(), nums.size());
        stack<int> st;
        for (int i = 0; i < nums.size(); ++i)
        {
            while (!st.empty() && nums[st.top()] >= nums[i])
            {
                right[st.top()] = i;
                st.pop();
            }
            if (!st.empty())
                left[i] = st.top();
            st.push(i);
        }
        long long result = 0;
        for (int i = 0; i < nums.size(); ++i)
        {
            int r = right[i] - 1, l = left[i] + 1;
            long long sum = preSum[r + 1] - preSum[l];
            result = max(result, (long long)nums[i] * sum);
        }
        return result % MOD;
    }
};