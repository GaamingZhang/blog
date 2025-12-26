class Solution
{
public:
    bool find132pattern(vector<int> &nums)
    {
        if (nums.size() < 3)
            return false;
        stack<int> st;
        int thirdValue = INT_MIN;
        for (int i = nums.size() - 1; i >= 0; --i)
        {
            if (nums[i] < thirdValue)
                return true;
            while (!st.empty() && nums[i] > st.top())
            {
                thirdValue = st.top();
                st.pop();
            }
            st.push(nums[i]);
        }
        return false;
    }
};