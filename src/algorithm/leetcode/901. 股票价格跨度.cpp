class StockSpanner
{
public:
    StockSpanner()
    {
    }

    int next(int price)
    {
        while (!st.empty() && price >= st.top().first)
        {
            st.pop();
        }
        int answer = st.empty() ? index : index - st.top().second;
        st.push({price, index++});
        return answer;
    }

private:
    int index = 1;
    // {data,index}
    stack<pair<int, int>> st;
};

/**
 * Your StockSpanner object will be instantiated and called as such:
 * StockSpanner* obj = new StockSpanner();
 * int param_1 = obj->next(price);
 */