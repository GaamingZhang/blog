class Solution
{
public:
    long long zeroFilledSubarray(vector<int> &nums)
    {
        long long ans = 0;
        int last = -1;
        for (int i = 0; i < nums.size(); ++i)
        {
            if (nums[i])
            {
                last = i;
            }
            else
            {
                ans += i - last;
            }
        }
        return ans;
    }
};