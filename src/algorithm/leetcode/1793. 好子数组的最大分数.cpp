class Solution
{
public:
    int maximumScore(vector<int> &nums, int k)
    {
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
            {
                left[i] = st.top();
            }
            st.push(i);
        }
        int ans = 0;
        for (int i = 0; i < nums.size(); ++i)
        {
            int l = left[i], r = right[i];
            if (l < k && k < r)
            {
                ans = max(ans, nums[i] * (r - l - 1));
            }
        }
        return ans;
    }
};