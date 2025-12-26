class Solution
{
public:
    int minOperations(vector<int> &nums)
    {
        stack<int> st;
        int result = 0;
        for (int i = 0; i < nums.size(); ++i)
        {
            while (!st.empty() && nums[st.top()] > nums[i])
            {
                ++result;
                st.pop();
            }
            if (nums[i] == 0 || (!st.empty() && nums[st.top()] == nums[i]))
                continue;
            st.push(i);
        }
        return result + st.size();
    }
};