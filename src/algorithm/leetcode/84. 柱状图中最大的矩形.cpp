class Solution
{
public:
    int largestRectangleArea(vector<int> &heights)
    {
        vector<int> left(heights.size(), -1), right(heights.size(), heights.size());
        stack<int> st;
        for (int i = 0; i < heights.size(); ++i)
        {
            while (!st.empty() && heights[st.top()] >= heights[i])
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
        for (int i = 0; i < heights.size(); ++i)
        {
            ans = max(ans, heights[i] * (right[i] - left[i] - 1));
        }
        return ans;
    }
};