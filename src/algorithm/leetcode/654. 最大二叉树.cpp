/**
 * Definition for a binary tree node.
 * struct TreeNode {
 *     int val;
 *     TreeNode *left;
 *     TreeNode *right;
 *     TreeNode() : val(0), left(nullptr), right(nullptr) {}
 *     TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
 *     TreeNode(int x, TreeNode *left, TreeNode *right) : val(x), left(left), right(right) {}
 * };
 */
class Solution {
public:
    TreeNode* constructMaximumBinaryTree(vector<int>& nums) {
        vector<int> left(nums.size(),-1),right(nums.size(),-1);
        vector<TreeNode*> tree(nums.size(),nullptr);
        stack<int> st;
        for(int i=0;i<nums.size();++i){
            tree[i]=new TreeNode(nums[i]);
            while(!st.empty() && nums[i]>nums[st.top()]){
                right[st.top()]=i;
                st.pop();
            }
            if(!st.empty()){
                left[i]=st.top();
            }
            st.push(i);
        }
        TreeNode* root=nullptr;
        for(int i=0;i<nums.size();++i){
            if(left[i]==-1 && right[i]==-1){
                root=tree[i];
            }else if(right[i]==-1 || (left[i]!=-1 && nums[left[i]]<nums[right[i]])){
                tree[left[i]]->right=tree[i];
            }else {
                tree[right[i]]->left=tree[i];
            }
        }
        return root;
    }
};