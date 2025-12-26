class Solution
{
public:
    int maximalRectangle(vector<vector<char>> &matrix)
    {
        if (matrix.empty())
            return 0;
        int m = matrix.size();
        int n = matrix[0].size();
        vector<int> height(n, 0);
        int maxA = 0;
        for (int i = 0; i < m; i++)
        {
            for (int j = 0; j < n; j++)
            {
                if (matrix[i][j] == '1')
                    height[j]++;
                else
                    height[j] = 0;
            }
            maxA = max(maxA, largestRectangleArea(height));
        }
        return maxA;
    }
    int largestRectangleArea(vector<int> &heights)
    {
        stack<int> st;
        vector<int> left(heights.size(), -1), right(heights.size(), heights.size());
        for (int i = 0; i < heights.size(); i++)
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
        int area = 0;
        for (int i = 0; i < heights.size(); ++i)
        {
            int l = left[i], r = right[i];
            area = max(area, heights[i] * (r - l - 1));
        }
        return area;
    }
};