class Solution
{
public:
    vector<int> dailyTemperatures(vector<int> &temperatures)
    {
        vector<int> answer(temperatures.size(), 0);
        stack<int> stIndex;
        for (int i = 0; i < temperatures.size(); ++i)
        {
            while (!stIndex.empty() && temperatures[i] > temperatures[stIndex.top()])
            {
                answer[stIndex.top()] = i - stIndex.top();
                stIndex.pop();
            }
            stIndex.push(i);
        }
        return answer;
    }
};