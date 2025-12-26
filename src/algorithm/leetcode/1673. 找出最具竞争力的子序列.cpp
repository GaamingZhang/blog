class Solution
{
public:
    vector<int> mostCompetitive(vector<int> &nums, int k)
    {
        int targetRemoveNum = max(0, (int)(nums.size() - k));
        vector<int> st;
        for (int i = 0; i < nums.size(); ++i)
        {
            while (!st.empty() && st.back() > nums[i] && targetRemoveNum)
            {
                st.pop_back();
                --targetRemoveNum;
            }
            st.push_back(nums[i]);
        }
        while (targetRemoveNum > 0)
        {
            st.pop_back();
            --targetRemoveNum;
        }
        return st;
    }
};