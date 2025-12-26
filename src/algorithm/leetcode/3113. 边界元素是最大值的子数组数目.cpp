class Solution
{
public:
    long long numberOfSubarrays(vector<int> &nums)
    {
        long long result = nums.size();
        // {data,count}
        stack<pair<int, long long>> st;
        for (int i = 0; i < nums.size(); ++i)
        {
            while (!st.empty() && nums[i] > st.top().first)
                st.pop();
            if (!st.empty() && st.top().first == nums[i])
            {
                result += st.top().second;
                st.top().second++;
            }
            else
            {
                st.push({nums[i], 1});
            }
        }
        return result;
    }
};