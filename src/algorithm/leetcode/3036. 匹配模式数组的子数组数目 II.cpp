class Solution
{
public:
    int countMatchingSubarrays(vector<int> &nums, vector<int> &pattern)
    {
        vector<int> next = getNext(pattern);
        int cnt = 0, result = 0;
        for (int i = 0; i < nums.size(); ++i)
        {
            while (cnt && !isOk(nums, pattern[cnt], i))
            {
                cnt = next[cnt - 1];
            }
            if (isOk(nums, pattern[cnt], i))
            {
                ++cnt;
            }
            if (cnt == pattern.size())
            {
                ++result;
                cnt = next[cnt - 1];
            }
        }
        return result;
    }
    bool isOk(const vector<int> &nums, int pattern, int pos)
    {
        if (pos + 1 == nums.size())
            return false;
        if (pattern == 1)
            return nums[pos + 1] > nums[pos];
        if (pattern == 0)
            return nums[pos + 1] == nums[pos];
        if (pattern == -1)
            return nums[pos + 1] < nums[pos];
        return false;
    }

    vector<int> getNext(const vector<int> &pattern)
    {
        int cnt = 0;
        vector<int> next(pattern.size(), 0);
        for (int i = 1; i < pattern.size(); ++i)
        {
            while (cnt && pattern[cnt] != pattern[i])
            {
                cnt = next[cnt - 1];
            }
            if (pattern[cnt] == pattern[i])
            {
                ++cnt;
            }
            next[i] = cnt;
        }
        return next;
    }
};