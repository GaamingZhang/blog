class Solution
{
public:
    vector<int> answerQueries(vector<int> &nums, vector<int> &queries)
    {
        sort(nums.begin(), nums.end(), less<int>());
        vector<int> preSum(nums.size(), nums[0]);
        for (int i = 1; i < nums.size(); ++i)
        {
            preSum[i] = preSum[i - 1] + nums[i];
        }
        vector<int> result(queries.size(), 0);
        for (int i = 0; i < queries.size(); ++i)
        {
            result[i] = upper_bound(preSum.begin(), preSum.end(), queries[i]) - preSum.begin();
        }
        return result;
    }
};