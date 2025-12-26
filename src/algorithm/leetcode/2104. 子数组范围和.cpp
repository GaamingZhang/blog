class Solution
{
public:
    long long subArrayRanges(vector<int> &nums)
    {
        vector<int> leftMax(nums.size(), -1), leftMin(nums.size(), -1);
        vector<int> rightMax(nums.size(), nums.size()), rightMin(nums.size(), nums.size());
        stack<int> maxSt, minSt;
        for (int i = 0; i < nums.size(); ++i)
        {
            // 最近一个大于当前数
            while (!maxSt.empty() && nums[maxSt.top()] <= nums[i])
            {
                rightMax[maxSt.top()] = i;
                maxSt.pop();
            }
            if (!maxSt.empty())
                leftMax[i] = maxSt.top();
            maxSt.push(i);

            // 最近一个小于当前数
            while (!minSt.empty() && nums[minSt.top()] >= nums[i])
            {
                rightMin[minSt.top()] = i;
                minSt.pop();
            }
            if (!minSt.empty())
                leftMin[i] = minSt.top();
            minSt.push(i);
        }

        long long result = 0;
        for (int i = 0; i < nums.size(); ++i)
        {
            result += (long long)(nums[i]) * (i - leftMax[i]) * (rightMax[i] - i);
            result -= (long long)(nums[i]) * (i - leftMin[i]) * (rightMin[i] - i);
        }
        return result;
    }
};